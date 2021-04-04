package scanner

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/scanner/metadata"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/pool"
)

type TagScanner struct {
	rootFolder  string
	ds          model.DataStore
	mapper      *mediaFileMapper
	plsSync     *playlistSync
	cnt         *counters
	cacheWarmer core.CacheWarmer
}

func NewTagScanner(rootFolder string, ds model.DataStore, cacheWarmer core.CacheWarmer) *TagScanner {
	return &TagScanner{
		rootFolder:  rootFolder,
		mapper:      newMediaFileMapper(rootFolder),
		plsSync:     newPlaylistSync(ds),
		ds:          ds,
		cacheWarmer: cacheWarmer,
	}
}

type (
	counters struct {
		added     int64
		updated   int64
		deleted   int64
		playlists int64
	}
	dirMap map[string]dirStats
)

const (
	// filesBatchSize used for batching file metadata extraction
	filesBatchSize = 100
)

// TagScanner algorithm overview:
// Load all directories from the DB
// Traverse the music folder, collecting each subfolder's ModTime (self or any non-dir children, whichever is newer)
// For each changed folder: get all files from DB whose path starts with the changed folder (non-recursively), check each file:
//	    if file in folder is newer, update the one in DB
//      if file in folder does not exists in DB, add it
// 	    for each file in the DB that is not found in the folder, delete it from DB
// Compare directories in the fs with the ones in the DB to find deleted folders
// For each deleted folder: delete all files from DB whose path starts with the delete folder path (non-recursively)
// Create new albums/artists, update counters:
//      collect all albumIDs and artistIDs from previous steps
//	    refresh the collected albums and artists with the metadata from the mediafiles
// For each changed folder, process playlists:
//      If the playlist is not in the DB, import it, setting sync = true
//      If the playlist is in the DB and sync == true, import it, or else skip it
// Delete all empty albums, delete all empty artists, clean-up playlists
func (s *TagScanner) Scan(ctx context.Context, lastModifiedSince time.Time, progress chan uint32) error {
	ctx = s.withAdminUser(ctx)
	start := time.Now()

	allDBDirs, err := s.getDBDirTree(ctx)
	if err != nil {
		return err
	}

	allFSDirs := dirMap{}
	var changedDirs []string
	s.cnt = &counters{}

	foldersFound, walkerError := s.getRootFolderWalker(ctx)
	for {
		folderStats, more := <-foldersFound
		if !more {
			break
		}
		progress <- folderStats.AudioFilesCount
		allFSDirs[folderStats.Path] = folderStats

		if s.folderHasChanged(ctx, folderStats, allDBDirs, lastModifiedSince) {
			changedDirs = append(changedDirs, folderStats.Path)
			log.Debug("Processing changed folder", "dir", folderStats.Path)
			err := s.processChangedDir(ctx, folderStats.Path)
			if err != nil {
				log.Error("Error updating folder in the DB", "dir", folderStats.Path, err)
			}
		}
	}

	if err := <-walkerError; err != nil {
		log.Error("Scan was interrupted by error. See errors above", err)
		return err
	}

	// If the media folder is empty, abort to avoid deleting all data
	if len(allFSDirs) <= 1 {
		log.Error(ctx, "Media Folder is empty. Aborting scan.", "folder", s.rootFolder)
		return nil
	}

	deletedDirs := s.getDeletedDirs(ctx, allFSDirs, allDBDirs)
	if len(deletedDirs)+len(changedDirs) == 0 {
		log.Debug(ctx, "No changes found in Music Folder", "folder", s.rootFolder, "elapsed", time.Since(start))
		return nil
	}

	for _, dir := range deletedDirs {
		err := s.processDeletedDir(ctx, dir)
		if err != nil {
			log.Error("Error removing deleted folder from DB", "dir", dir, err)
		}
	}

	s.cnt.playlists = 0
	if conf.Server.AutoImportPlaylists {
		// Now that all mediafiles are imported/updated, search for and import/update playlists
		u, _ := request.UserFrom(ctx)
		for _, dir := range changedDirs {
			info := allFSDirs[dir]
			if info.HasPlaylist {
				if !u.IsAdmin {
					log.Warn("Playlists will not be imported, as there are no admin users yet, "+
						"Please create an admin user first, and then update the playlists for them to be imported", "dir", dir)
				} else {
					s.cnt.playlists = s.plsSync.processPlaylists(ctx, dir)
				}
			}
		}
	} else {
		log.Debug("Playlist auto-import is disabled")
	}

	err = s.ds.GC(log.NewContext(ctx), s.rootFolder)
	log.Info("Finished processing Music Folder", "folder", s.rootFolder, "elapsed", time.Since(start),
		"added", s.cnt.added, "updated", s.cnt.updated, "deleted", s.cnt.deleted, "playlistsImported", s.cnt.playlists)

	return err
}

func (s *TagScanner) getRootFolderWalker(ctx context.Context) (walkResults, chan error) {
	start := time.Now()
	log.Trace(ctx, "Loading directory tree from music folder", "folder", s.rootFolder)
	results := make(chan dirStats, 5000)
	walkerError := make(chan error)
	go func() {
		err := walkDirTree(ctx, s.rootFolder, results)
		if err != nil {
			log.Error("There were errors reading directories from filesystem", err)
		}
		walkerError <- err
		log.Debug("Finished reading directories from filesystem", "elapsed", time.Since(start))
	}()
	return results, walkerError
}

func (s *TagScanner) getDBDirTree(ctx context.Context) (map[string]struct{}, error) {
	start := time.Now()
	log.Trace(ctx, "Loading directory tree from database", "folder", s.rootFolder)

	repo := s.ds.MediaFile(ctx)
	dirs, err := repo.FindPathsRecursively(s.rootFolder)
	if err != nil {
		return nil, err
	}
	resp := map[string]struct{}{}
	for _, d := range dirs {
		resp[filepath.Clean(d)] = struct{}{}
	}

	log.Debug("Directory tree loaded from DB", "total", len(resp), "elapsed", time.Since(start))
	return resp, nil
}

func (s *TagScanner) folderHasChanged(ctx context.Context, folder dirStats, dbDirs map[string]struct{}, lastModified time.Time) bool {
	_, inDB := dbDirs[folder.Path]
	// If is a new folder with at least one song OR it was modified after lastModified
	return (!inDB && (folder.AudioFilesCount > 0)) || folder.ModTime.After(lastModified)
}

func (s *TagScanner) getDeletedDirs(ctx context.Context, fsDirs dirMap, dbDirs map[string]struct{}) []string {
	start := time.Now()
	log.Trace(ctx, "Checking for deleted folders")
	var deleted []string

	for d := range dbDirs {
		if _, ok := fsDirs[d]; !ok {
			deleted = append(deleted, d)
		}
	}

	sort.Strings(deleted)
	log.Debug(ctx, "Finished deleted folders check", "total", len(deleted), "elapsed", time.Since(start))
	return deleted
}

func (s *TagScanner) processDeletedDir(ctx context.Context, dir string) error {
	start := time.Now()
	buffer := newRefreshBuffer(ctx, s.ds)

	mfs, err := s.ds.MediaFile(ctx).FindAllByPath(dir)
	if err != nil {
		return err
	}

	c, err := s.ds.MediaFile(ctx).DeleteByPath(dir)
	if err != nil {
		return err
	}
	s.cnt.deleted += c

	for _, t := range mfs {
		buffer.accumulate(t)
		s.cacheWarmer.AddAlbum(ctx, t.AlbumID)
	}

	err = buffer.flush()
	log.Info(ctx, "Finished processing deleted folder", "dir", dir, "purged", len(mfs), "elapsed", time.Since(start))
	return err
}

func (s *TagScanner) processChangedDir(ctx context.Context, dir string) error {
	start := time.Now()
	buffer := newRefreshBuffer(ctx, s.ds)

	// Load folder's current tracks from DB into a map
	currentTracks := map[string]model.MediaFile{}
	ct, err := s.ds.MediaFile(ctx).FindAllByPath(dir)
	if err != nil {
		return err
	}
	for _, t := range ct {
		currentTracks[t.Path] = t
	}

	// Load tracks FileInfo from the folder
	files, err := loadAllAudioFiles(dir)
	if err != nil {
		return err
	}

	// If no files to process, return
	if len(files)+len(currentTracks) == 0 {
		return nil
	}

	orphanTracks := map[string]model.MediaFile{}
	for k, v := range currentTracks {
		orphanTracks[k] = v
	}

	// If track from folder is newer than the one in DB, select for update/insert in DB
	log.Trace(ctx, "Processing changed folder", "dir", dir, "tracksInDB", len(currentTracks), "tracksInFolder", len(files))
	var filesToUpdate []string
	for filePath, info := range files {
		c, ok := currentTracks[filePath]
		if !ok {
			filesToUpdate = append(filesToUpdate, filePath)
			s.cnt.added++
		}
		if ok && info.ModTime().After(c.UpdatedAt) {
			filesToUpdate = append(filesToUpdate, filePath)
			s.cnt.updated++
		}

		// Force a refresh of the album and artist, to cater for cover art files
		buffer.accumulate(c)

		// Only leaves in orphanTracks the ones not found in the folder. After this loop any remaining orphanTracks
		// are considered gone from the music folder and will be deleted from DB
		delete(orphanTracks, filePath)
	}

	numUpdatedTracks := 0
	numPurgedTracks := 0

	if len(filesToUpdate) > 0 {
		numUpdatedTracks, err = s.addOrUpdateTracksInDB(ctx, dir, currentTracks, filesToUpdate, buffer)
		if err != nil {
			return err
		}
	}

	if len(orphanTracks) > 0 {
		numPurgedTracks, err = s.deleteOrphanSongs(ctx, dir, orphanTracks, buffer)
		if err != nil {
			return err
		}
	}

	// Pre cache all changed album artwork
	for albumID := range buffer.album {
		s.cacheWarmer.AddAlbum(ctx, albumID)
	}

	err = buffer.flush()
	log.Info(ctx, "Finished processing changed folder", "dir", dir, "updated", numUpdatedTracks,
		"purged", numPurgedTracks, "elapsed", time.Since(start))
	return err
}

func (s *TagScanner) deleteOrphanSongs(ctx context.Context, dir string, tracksToDelete map[string]model.MediaFile, buffer *refreshBuffer) (int, error) {
	numPurgedTracks := 0

	log.Debug(ctx, "Deleting orphan tracks from DB", "dir", dir, "numTracks", len(tracksToDelete))
	// Remaining tracks from DB that are not in the folder are deleted
	for _, ct := range tracksToDelete {
		numPurgedTracks++
		buffer.accumulate(ct)
		if err := s.ds.MediaFile(ctx).Delete(ct.ID); err != nil {
			return 0, err
		}
		s.cnt.deleted++
	}
	return numPurgedTracks, nil
}

type artistItem struct {
	artistId string
}

func (s *TagScanner) execute(workload interface{}) {
	es := core.NewExternalMetadata(s.ds)
	ctx := context.Background()
	item := workload.(artistItem)
	_, err := es.UpdateArtistInfo(ctx, item.artistId, 0, false)
	if err != nil {
		log.Warn(err.Error())
		return
	}
}

func (s *TagScanner) addOrUpdateTracksInDB(ctx context.Context, dir string, currentTracks map[string]model.MediaFile, filesToUpdate []string, buffer *refreshBuffer) (int, error) {
	numUpdatedTracks := 0

	p, err := pool.NewPool("artistInfo", 10, s.execute)

	if err != nil {
		log.Error(context.Background(), "Error creating pool for Artist Info", err)
	}

	log.Trace(ctx, "Updating mediaFiles in DB", "dir", dir, "numFiles", len(filesToUpdate))
	// Break the file list in chunks to avoid calling ffmpeg with too many parameters
	chunks := utils.BreakUpStringSlice(filesToUpdate, filesBatchSize)
	for _, chunk := range chunks {
		// Load tracks Metadata from the folder
		newTracks, err := s.loadTracks(chunk)
		if err != nil {
			return 0, err
		}

		// If track from folder is newer than the one in DB, update/insert in DB
		log.Trace(ctx, "Updating mediaFiles in DB", "dir", dir, "files", chunk, "numFiles", len(chunk))
		for i := range newTracks {
			n := newTracks[i]

			// Keep current annotations if the track is in the DB
			if t, ok := currentTracks[n.Path]; ok {
				n.Annotations = t.Annotations
			}
			err := s.ds.MediaFile(ctx).Put(&n)
			if err != nil {
				return 0, err
			}
			buffer.accumulate(n)
			numUpdatedTracks++

			p.Submit(artistItem{artistId: n.ArtistID})
		}
	}
	return numUpdatedTracks, nil
}

func (s *TagScanner) loadTracks(filePaths []string) (model.MediaFiles, error) {
	mds, err := metadata.Extract(filePaths...)
	if err != nil {
		return nil, err
	}

	var mfs model.MediaFiles
	for _, md := range mds {
		mf := s.mapper.toMediaFile(md)
		mfs = append(mfs, mf)
	}
	return mfs, nil
}

func (s *TagScanner) withAdminUser(ctx context.Context) context.Context {
	u, err := s.ds.User(ctx).FindFirstAdmin()
	if err != nil {
		log.Warn(ctx, "No admin user found!", err)
		u = &model.User{}
	}

	ctx = request.WithUsername(ctx, u.UserName)
	return request.WithUser(ctx, *u)
}

func loadAllAudioFiles(dirPath string) (map[string]os.FileInfo, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}
	files, err := dir.Readdir(-1)
	if err != nil {
		return nil, err
	}
	audioFiles := make(map[string]os.FileInfo)
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if strings.HasPrefix(f.Name(), ".") {
			continue
		}
		filePath := filepath.Join(dirPath, f.Name())
		if !utils.IsAudioFile(filePath) {
			continue
		}
		fi, err := os.Stat(filePath)
		if err != nil {
			log.Error("Could not stat file", "filePath", filePath, err)
		} else {
			audioFiles[filePath] = fi
		}
	}

	return audioFiles, nil
}

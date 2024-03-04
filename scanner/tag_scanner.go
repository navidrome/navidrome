package scanner

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/scanner/metadata"
	"github.com/navidrome/navidrome/scanner/metadata/cuesheet"
	_ "github.com/navidrome/navidrome/scanner/metadata/ffmpeg"
	_ "github.com/navidrome/navidrome/scanner/metadata/taglib"
	"github.com/navidrome/navidrome/utils/slice"
)

type TagScanner struct {
	rootFolder  string
	ds          model.DataStore
	plsSync     *playlistImporter
	cnt         *counters
	mapper      *MediaFileMapper
	cacheWarmer artwork.CacheWarmer
	cueCache    map[string]*cuesheet.Cuesheet
}

func NewTagScanner(rootFolder string, ds model.DataStore, playlists core.Playlists, cacheWarmer artwork.CacheWarmer) FolderScanner {
	s := &TagScanner{
		rootFolder:  rootFolder,
		plsSync:     newPlaylistImporter(ds, playlists, cacheWarmer, rootFolder),
		ds:          ds,
		cacheWarmer: cacheWarmer,
		cueCache:    map[string]*cuesheet.Cuesheet{},
	}
	metadata.LogExtractors()

	return s
}

type dirMap map[string]dirStats

type counters struct {
	added     int64
	updated   int64
	deleted   int64
	playlists int64
}

func (cnt *counters) total() int64 { return cnt.added + cnt.updated + cnt.deleted }

const (
	// filesBatchSize used for batching file metadata extraction
	filesBatchSize = 100
)

// Scan algorithm overview:
// Load all directories from the DB
// Traverse the music folder, collecting each subfolder's ModTime (self or any non-dir children, whichever is newer)
// For each changed folder: get all files from DB whose path starts with the changed folder (non-recursively), check each file:
// - if file in folder is newer, update the one in DB
// - if file in folder does not exists in DB, add it
// - for each file in the DB that is not found in the folder, delete it from DB
// Compare directories in the fs with the ones in the DB to find deleted folders
// For each deleted folder: delete all files from DB whose path starts with the delete folder path (non-recursively)
// Create new albums/artists, update counters:
// - collect all albumIDs and artistIDs from previous steps
// - refresh the collected albums and artists with the metadata from the mediafiles
// For each changed folder, process playlists:
// - If the playlist is not in the DB, import it, setting sync = true
// - If the playlist is in the DB and sync == true, import it, or else skip it
// Delete all empty albums, delete all empty artists, clean-up playlists
func (s *TagScanner) Scan(ctx context.Context, lastModifiedSince time.Time, progress chan uint32) (int64, error) {
	ctx = auth.WithAdminUser(ctx, s.ds)
	start := time.Now()

	// Special case: if lastModifiedSince is zero, re-import all files
	fullScan := lastModifiedSince.IsZero()

	// If the media folder is empty (no music and no subfolders), abort to avoid deleting all data from DB
	empty, err := isDirEmpty(ctx, s.rootFolder)
	if err != nil {
		return 0, err
	}
	if empty && !fullScan {
		log.Error(ctx, "Media Folder is empty. Aborting scan.", "folder", s.rootFolder)
		return 0, nil
	}

	allDBDirs, err := s.getDBDirTree(ctx)
	if err != nil {
		return 0, err
	}

	allFSDirs := dirMap{}
	var changedDirs []string
	s.cnt = &counters{}
	genres := newCachedGenreRepository(ctx, s.ds.Genre(ctx))
	s.mapper = NewMediaFileMapper(s.rootFolder, genres)
	refresher := newRefresher(s.ds, s.cacheWarmer, allFSDirs)

	log.Trace(ctx, "Loading directory tree from music folder", "folder", s.rootFolder)
	foldersFound, walkerError := walkDirTree(ctx, s.rootFolder)

	for {
		folderStats, more := <-foldersFound
		if !more {
			break
		}
		progress <- folderStats.AudioFilesCount
		allFSDirs[folderStats.Path] = folderStats

		if s.folderHasChanged(folderStats, allDBDirs, lastModifiedSince) {
			changedDirs = append(changedDirs, folderStats.Path)
			log.Debug("Processing changed folder", "dir", folderStats.Path)
			err := s.processChangedDir(ctx, refresher, fullScan, folderStats.Path)
			if err != nil {
				log.Error("Error updating folder in the DB", "dir", folderStats.Path, err)
			}
		}
	}

	if err := <-walkerError; err != nil {
		log.Error("Scan was interrupted by error. See errors above", err)
		return 0, err
	}

	deletedDirs := s.getDeletedDirs(ctx, allFSDirs, allDBDirs)
	if len(deletedDirs)+len(changedDirs) == 0 {
		log.Debug(ctx, "No changes found in Music Folder", "folder", s.rootFolder, "elapsed", time.Since(start))
		return 0, nil
	}

	for _, dir := range deletedDirs {
		err := s.processDeletedDir(ctx, refresher, dir)
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

	return s.cnt.total(), err
}

func isDirEmpty(ctx context.Context, dir string) (bool, error) {
	children, stats, err := loadDir(ctx, dir)
	if err != nil {
		return false, err
	}
	return len(children) == 0 && stats.AudioFilesCount == 0, nil
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

func (s *TagScanner) folderHasChanged(folder dirStats, dbDirs map[string]struct{}, lastModified time.Time) bool {
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

func (s *TagScanner) processDeletedDir(ctx context.Context, refresher *refresher, dir string) error {
	start := time.Now()

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
		refresher.accumulate(t)
	}

	err = refresher.flush(ctx)
	log.Info(ctx, "Finished processing deleted folder", "dir", dir, "purged", len(mfs), "elapsed", time.Since(start))
	return err
}

func (s *TagScanner) addMediaForUpdate(filesToUpdate []string, filePath string, visitedPaths map[string]struct{}) []string {
	if _, visited := visitedPaths[filePath]; !visited {
		s.cnt.added++
		return append(filesToUpdate, filePath)
	}
	return filesToUpdate
}

func (s *TagScanner) removeFromOrphan(orphanTracks map[string]model.MediaFile, tracks []model.MediaFile) {
	for _, track := range tracks {
		delete(orphanTracks, track.ID)
	}
}

func (s *TagScanner) processChangedDir(ctx context.Context, refresher *refresher, fullScan bool, dir string) error {
	start := time.Now()

	// Tracks for delete after scan
	orphanTracks := map[string]model.MediaFile{}

	// Load folder's current tracks from DB into a map
	currentTracks := map[string]model.MediaFiles{}
	ct, err := s.ds.MediaFile(ctx).FindAllByPath(dir)
	if err != nil {
		return err
	}
	for _, t := range ct {
		currentTracks[t.Path] = append(currentTracks[t.Path], t)
		// We don't need a full MediaFile here
		orphanTracks[t.ID] = model.MediaFile{
			AlbumID:       t.AlbumID,
			AlbumArtistID: t.AlbumArtistID,
		}
	}

	// Load track list from the folder
	files, err := loadAllAudioFiles(dir, conf.Server.Scanner.CUESheetSupport)
	if err != nil {
		return err
	}

	// If no files to process, return
	if len(files)+len(currentTracks) == 0 {
		return nil
	}

	// If track from folder is newer than the one in DB, select for update/insert in DB
	log.Trace(ctx, "Processing changed folder", "dir", dir, "tracksInDB", len(currentTracks), "tracksInFolder", len(files))
	var filesToUpdate []string
	visitedPaths := map[string]struct{}{}

	// Handle each media file from folder
	handleMediaPath := func(mediaPath string, trackCallback func(fileModTime time.Time, trackFromDB *model.MediaFile) bool) bool {
		tracks, inDB := currentTracks[mediaPath]
		info, err := os.Stat(mediaPath)
		if err != nil {
			log.Error("Could not stat media file", "mediaPath", mediaPath, err)
			return false
		}

		if !inDB || fullScan {
			filesToUpdate = s.addMediaForUpdate(filesToUpdate, mediaPath, visitedPaths)
		} else {
			trackFromDB := tracks[0]
			// Need add only one from multi-track source, but mark as visited all tracks
			if trackCallback(info.ModTime(), &trackFromDB) {
				filesToUpdate = s.addMediaForUpdate(filesToUpdate, mediaPath, visitedPaths)
			} else {
				s.removeFromOrphan(orphanTracks, tracks)
			}

			// Force a refresh of the album and artist, to cater for cover art files
			refresher.accumulate(trackFromDB)
		}
		return true
	}

	for filePath, entry := range files {
		if model.IsCueSheetFile(filePath) {
			cueInfo, err := entry.Info()
			if err != nil {
				log.Error("Could not stat CUE file", "filePath", filePath, err)
				continue
			}

			cue, err := cuesheet.ReadFromFile(filePath)
			if err != nil {
				log.Error("Could not read CUE file", "filePath", filePath, err)
				continue
			}
			for _, f := range cue.File {
				mediaPath := filepath.Join(filepath.Dir(filePath), filepath.Base(f.FileName))
				if handleMediaPath(mediaPath, func(fileModTime time.Time, trackFromDB *model.MediaFile) bool {
					return cueInfo.ModTime().After(trackFromDB.UpdatedAt) ||
						(conf.Server.Scanner.CUESheetSupport != consts.CUEExternal &&
							fileModTime.After(trackFromDB.UpdatedAt))
				}) {
					// Store CUE data in cache for future use
					s.cueCache[filePath] = cue
				}
			}
		} else {
			handleMediaPath(filePath, func(fileModTime time.Time, trackFromDB *model.MediaFile) bool {
				return fileModTime.After(trackFromDB.UpdatedAt)
			})
		}
	}

	numUpdatedTracks := 0
	numPurgedTracks := 0

	if len(filesToUpdate) > 0 {
		numUpdatedTracks, err = s.addOrUpdateTracksInDB(ctx, refresher, dir, filesToUpdate, func(track *model.MediaFile) {
			// Keep current annotations if the track is in the DB
			if tracks, ok := currentTracks[track.Path]; ok {
				for _, dbTrack := range tracks {
					if dbTrack.ID == track.ID {
						track.Annotations = dbTrack.Annotations
						track.Bookmarkable = dbTrack.Bookmarkable
						// Track will be updated in DB, no need to remove it
						delete(orphanTracks, track.ID)
						break
					}
				}
			}
		})
		if err != nil {
			return err
		}
	}

	if len(orphanTracks) > 0 {
		numPurgedTracks, err = s.deleteOrphanSongs(ctx, refresher, dir, orphanTracks)
		if err != nil {
			return err
		}
	}

	err = refresher.flush(ctx)
	log.Info(ctx, "Finished processing changed folder", "dir", dir, "updated", numUpdatedTracks,
		"deleted", numPurgedTracks, "elapsed", time.Since(start))
	return err
}

func (s *TagScanner) deleteOrphanSongs(
	ctx context.Context,
	refresher *refresher,
	dir string,
	tracksToDelete map[string]model.MediaFile,
) (int, error) {
	numPurgedTracks := 0

	log.Debug(ctx, "Deleting orphan tracks from DB", "dir", dir, "numTracks", len(tracksToDelete))
	// Remaining tracks from DB that are not in the folder are deleted
	for _, ct := range tracksToDelete {
		numPurgedTracks++
		refresher.accumulate(ct)
		if err := s.ds.MediaFile(ctx).Delete(ct.ID); err != nil {
			return 0, err
		}
		s.cnt.deleted++
	}
	return numPurgedTracks, nil
}

func (s *TagScanner) addOrUpdateTracksInDB(
	ctx context.Context,
	refresher *refresher,
	dir string,
	filesToUpdate []string,
	trackCallback func(track *model.MediaFile)) (int, error) {
	numUpdatedTracks := 0

	log.Trace(ctx, "Updating mediaFiles in DB", "dir", dir, "numFiles", len(filesToUpdate))
	// Break the file list in chunks to avoid calling ffmpeg with too many parameters
	chunks := slice.BreakUp(filesToUpdate, filesBatchSize)
	for _, chunk := range chunks {
		// Load tracks Metadata from the folder
		newTracks, err := s.loadTracks(ctx, chunk)
		if err != nil {
			return 0, err
		}

		// If track from folder is newer than the one in DB, update/insert in DB
		log.Trace(ctx, "Updating mediaFiles in DB", "dir", dir, "files", chunk, "numFiles", len(chunk))
		for i := range newTracks {
			n := newTracks[i]
			trackCallback(&n)
			err := s.ds.MediaFile(ctx).Put(&n)
			if err != nil {
				return 0, err
			}
			refresher.accumulate(n)
			numUpdatedTracks++
		}
	}
	return numUpdatedTracks, nil
}

func (s *TagScanner) loadCueSheet(ctx context.Context, md metadata.Tags) model.MediaFiles {
	cueStr := md.CueSheet()
	if cueStr == "" || conf.Server.Scanner.CUESheetSupport == consts.CUEDisable {
		if cueStr != "" {
			log.Trace(ctx, "CUE sheet support disabled skip track", "filePath", md.FilePath())
		}
		return nil
	}
	extractor, err := cuesheet.NewExtractor(&md)
	if err != nil {
		log.Error(ctx, "Can't create CUE tags extractor", "filePath", md.FilePath(), err)
	}

	modes := strings.Split(conf.Server.Scanner.CUESheetSupport, ",")
	for i, mode := range modes {
		isLast := i == len(modes)-1
		switch strings.TrimSpace(strings.ToLower(mode)) {
		case consts.CUEEmbedded:
			cue, err := cuesheet.ReadCue(bytes.NewBuffer([]byte(cueStr)))
			if err != nil {
				log.Error(ctx, "Can't read embedded CUE", "filePath", md.FilePath(), err)
			}
			if err := extractor.Extract(cue, isLast, true); err != nil {
				log.Error(ctx, "Can't extract tags from embedded CUE", "filePath", md.FilePath(), err)
			}
		case consts.CUEExternal:
			if cue, ok := s.cueCache[md.FilePath()]; ok {
				if err := extractor.Extract(cue, isLast, false); err != nil {
					log.Error(ctx, "Can't extract tags from external CUE", "filePath", md.FilePath(), err)
				}
			}
		}
	}

	var mfs model.MediaFiles
	extractor.ForEachTrack(func(md metadata.Tags) {
		mfs = append(mfs, s.mapper.ToMediaFile(md))
	})
	return mfs
}

func (s *TagScanner) loadTracks(ctx context.Context, filePaths []string) (model.MediaFiles, error) {
	mds, err := metadata.Extract(filePaths...)
	if err != nil {
		return nil, err
	}

	var mfs model.MediaFiles
	for _, md := range mds {
		tracks := s.loadCueSheet(ctx, md)
		if len(tracks) > 0 {
			mfs = append(mfs, tracks...)
		} else {
			mf := s.mapper.ToMediaFile(md)
			mfs = append(mfs, mf)
		}
	}
	return mfs, nil
}

func loadAllAudioFiles(dirPath string, cueSupport string) (map[string]fs.DirEntry, error) {
	files, err := fs.ReadDir(os.DirFS(dirPath), ".")
	if err != nil {
		return nil, err
	}
	fileInfos := make(map[string]fs.DirEntry)
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if strings.HasPrefix(f.Name(), ".") {
			continue
		}
		filePath := filepath.Join(dirPath, f.Name())
		if model.IsCueSheetFile(filePath) {
			if strings.Contains(strings.ToLower(cueSupport), consts.CUEExternal) {
				fileInfos[filePath] = f
			}
			continue
		}
		if !model.IsAudioFile(filePath) {
			continue
		}
		fileInfos[filePath] = f
	}

	return fileInfos, nil
}

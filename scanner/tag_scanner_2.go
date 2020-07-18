package scanner

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/utils"
)

type TagScanner2 struct {
	rootFolder string
	ds         model.DataStore
	mapper     *mediaFileMapper
	plsSync    *playlistSync
	albumMap   *flushableMap
	artistMap  *flushableMap
	cnt        *counters
}

func NewTagScanner2(rootFolder string, ds model.DataStore) *TagScanner2 {
	return &TagScanner2{
		rootFolder: rootFolder,
		mapper:     newMediaFileMapper(rootFolder),
		plsSync:    newPlaylistSync(ds),
		ds:         ds,
	}
}

// Scan algorithm overview:
// Load all directories under the music folder, with their ModTime (self or any non-dir children)
// Find changed folders (based on lastModifiedSince) and deletes folders (comparing to the DB)
// For each deleted folder: delete all files from DB whose path starts with the delete folder path
// For each changed folder: Get all files from DB whose path starts with the changed folder, scan each file:
//	    if file in folder is newer, update the one in DB
//      if file in folder does not exists in DB, add
// 	    for each file in the DB that is not found in the folder, delete from DB
// Create new albums/artists, update counters:
//      collect all albumIDs and artistIDs from previous steps
//	    refresh the collected albums and artists with the metadata from the mediafiles
// Delete all empty albums, delete all empty Artists
func (s *TagScanner2) Scan(ctx context.Context, lastModifiedSince time.Time) error {
	start := time.Now()
	allDirs, err := s.getDirTree(ctx)
	if err != nil {
		return err
	}

	changedDirs := s.getChangedDirs(ctx, allDirs, lastModifiedSince)
	if len(changedDirs) == 0 {
		log.Debug(ctx, "No changes found in Music Folder", "folder", s.rootFolder)
		return nil
	}
	deletedDirs, _ := s.getDeletedDirs(ctx, allDirs, changedDirs)

	if log.CurrentLevel() >= log.LevelTrace {
		log.Info(ctx, "Folder changes detected", "changedFolders", len(changedDirs), "deletedFolders", len(deletedDirs),
			"changed", strings.Join(changedDirs, ";"), "deleted", strings.Join(deletedDirs, ";"))
	} else {
		log.Info(ctx, "Folder changes detected", "changedFolders", len(changedDirs), "deletedFolders", len(deletedDirs))
	}

	s.albumMap = newFlushableMap(ctx, "album", s.ds.Album(ctx).Refresh)
	s.artistMap = newFlushableMap(ctx, "artist", s.ds.Artist(ctx).Refresh)
	s.cnt = &counters{}

	for _, dir := range deletedDirs {
		err := s.processDeletedDir(ctx, dir)
		if err != nil {
			log.Error("Error removing deleted folder from DB", "path", dir, err)
		}
	}
	for _, dir := range changedDirs {
		err := s.processChangedDir(ctx, dir)
		if err != nil {
			log.Error("Error updating folder in the DB", "path", dir, err)
		}
	}

	_ = s.albumMap.flush()
	_ = s.artistMap.flush()

	// Now that all mediafiles are imported/updated, search for and import playlists
	for _, dir := range changedDirs {
		_ = s.plsSync.processPlaylists(ctx, dir)
	}

	err = s.ds.GC(log.NewContext(ctx))
	log.Info("Finished processing Music Folder", "folder", s.rootFolder, "elapsed", time.Since(start),
		"added", s.cnt.added, "updated", s.cnt.updated, "deleted", s.cnt.deleted)

	return err
}

func (s *TagScanner2) getDirTree(ctx context.Context) (dirMap, error) {
	start := time.Now()
	log.Trace(ctx, "Loading directory tree from music folder", "folder", s.rootFolder)
	dirs, err := loadDirTree(ctx, s.rootFolder)
	if err != nil {
		return nil, err
	}
	log.Debug("Directory tree loaded", "total", len(dirs), "elapsed", time.Since(start))
	return dirs, nil
}

func (s *TagScanner2) getChangedDirs(ctx context.Context, dirs dirMap, lastModified time.Time) []string {
	start := time.Now()
	log.Trace(ctx, "Checking for changed folders")
	var changed []string
	for d, t := range dirs {
		if t.After(lastModified) {
			changed = append(changed, d)
		}
	}
	sort.Strings(changed)
	log.Debug(ctx, "Finished changed folders check", "total", len(changed), "elapsed", time.Since(start))
	return changed
}

func (s *TagScanner2) getDeletedDirs(ctx context.Context, allDirs dirMap, changedDirs []string) ([]string, error) {
	start := time.Now()
	log.Trace(ctx, "Checking for deleted folders")

	var deleted []string
	repo := s.ds.MediaFile(ctx)

	// If rootFolder is in the list of changedDirs, optimize and only do one query to the DB
	var foldersToCheck []string
	if utils.StringInSlice(s.rootFolder, changedDirs) {
		foldersToCheck = []string{s.rootFolder}
	} else {
		foldersToCheck = changedDirs
	}

	for _, changedDir := range foldersToCheck {
		dirs, err := repo.FindPathsRecursively(changedDir)
		if err != nil {
			log.Error("Error getting subfolders from DB", "path", changedDir, err)
			continue
		}
		for _, d := range dirs {
			d := filepath.Clean(d)
			if _, ok := allDirs[d]; !ok {
				deleted = append(deleted, d)
			}
		}
	}

	sort.Strings(deleted)
	log.Debug(ctx, "Finished deleted folders check", "total", len(deleted), "elapsed", time.Since(start))
	return deleted, nil
}

func (s *TagScanner2) processDeletedDir(ctx context.Context, dir string) error {
	start := time.Now()

	mfs, err := s.ds.MediaFile(ctx).FindAllByPath(dir)
	if err != nil {
		return err
	}
	for _, t := range mfs {
		err = s.albumMap.update(t.AlbumID)
		if err != nil {
			return err
		}
		err = s.artistMap.update(t.AlbumArtistID)
		if err != nil {
			return err
		}
	}

	log.Info(ctx, "Finished processing deleted folder", "path", dir, "purged", len(mfs), "elapsed", time.Since(start))
	c, err := s.ds.MediaFile(ctx).DeleteByPath(dir)
	s.cnt.deleted += c
	return err
}

func (s *TagScanner2) processChangedDir(ctx context.Context, dir string) error {
	start := time.Now()

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
	files, err := LoadAllAudioFiles(dir)
	if err != nil {
		return err
	}

	// If no files to process, return
	if len(files)+len(currentTracks) == 0 {
		return nil
	}

	// If track from folder is newer than the one in DB, select for update/insert in DB and delete from the current tracks
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
		delete(currentTracks, filePath)

		// Force a refresh of the album and artist, to cater for cover art files
		err = s.albumMap.update(c.AlbumID)
		if err != nil {
			return err
		}
		err = s.artistMap.update(c.AlbumArtistID)
		if err != nil {
			return err
		}
	}

	numUpdatedTracks := 0
	numPurgedTracks := 0

	if len(filesToUpdate) > 0 {
		numUpdatedTracks, err = s.addOrUpdateTracksInDB(ctx, dir, filesToUpdate)
		if err != nil {
			return err
		}
	}

	if len(currentTracks) > 0 {
		numPurgedTracks, err = s.deleteOrphanSongs(ctx, dir, currentTracks)
		if err != nil {
			return err
		}
	}

	log.Info(ctx, "Finished processing changed folder", "dir", dir, "updated", numUpdatedTracks, "purged", numPurgedTracks, "elapsed", time.Since(start))
	return nil
}

func (s *TagScanner2) deleteOrphanSongs(ctx context.Context, dir string, tracksToDelete map[string]model.MediaFile) (int, error) {
	numPurgedTracks := 0

	log.Debug(ctx, "Deleting orphan tracks from DB", "dir", dir, "numTracks", len(tracksToDelete))
	// Remaining tracks from DB that are not in the folder are deleted
	for _, ct := range tracksToDelete {
		numPurgedTracks++
		err := s.albumMap.update(ct.AlbumID)
		if err != nil {
			return 0, err
		}
		err = s.artistMap.update(ct.AlbumArtistID)
		if err != nil {
			return 0, err
		}
		if err := s.ds.MediaFile(ctx).Delete(ct.ID); err != nil {
			return 0, err
		}
		s.cnt.deleted++
	}
	return numPurgedTracks, nil
}

func (s *TagScanner2) addOrUpdateTracksInDB(ctx context.Context, dir string, filesToUpdate []string) (int, error) {
	numUpdatedTracks := 0

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
			err := s.ds.MediaFile(ctx).Put(&n)
			if err != nil {
				return 0, err
			}
			err = s.albumMap.update(n.AlbumID)
			if err != nil {
				return 0, err
			}
			err = s.artistMap.update(n.AlbumArtistID)
			if err != nil {
				return 0, err
			}
			numUpdatedTracks++
		}
	}
	return numUpdatedTracks, nil
}

func (s *TagScanner2) loadTracks(filePaths []string) (model.MediaFiles, error) {
	mds, err := ExtractAllMetadata(filePaths)
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

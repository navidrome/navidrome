package scanner

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/model/request"
	"github.com/deluan/navidrome/utils"
)

type TagScanner2 struct {
	rootFolder string
	ds         model.DataStore
	mapper     *mediaFileMapper
	plsSync    *playlistSync
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
// Load all directories under the music folder, with their ModTime (self or any non-dir children, whichever is newer)
// Load all directories from the DB
// Compare both collections to find changed folders (based on lastModifiedSince) and deleted folders
// For each deleted folder: delete all files from DB whose path starts with the delete folder path (non-recursively)
// For each changed folder: get all files from DB whose path starts with the changed folder (non-recursively), check each file:
//	    if file in folder is newer, update the one in DB
//      if file in folder does not exists in DB, add it
// 	    for each file in the DB that is not found in the folder, delete it from DB
// Create new albums/artists, update counters:
//      collect all albumIDs and artistIDs from previous steps
//	    refresh the collected albums and artists with the metadata from the mediafiles
// For each changed folder, process playlists:
//      If the playlist is not in the DB, import it, setting sync = true
//      If the playlist is in the DB and sync == true, import it, or else skip it
// Delete all empty albums, delete all empty artists, clean-up playlists
func (s *TagScanner2) Scan(ctx context.Context, lastModifiedSince time.Time) error {
	ctx = s.withAdminUser(ctx)

	start := time.Now()
	allDirs, err := s.getDirTree(ctx)
	if err != nil {
		return err
	}

	allDBDirs, err := s.getDBDirTree(ctx)
	if err != nil {
		return err
	}

	changedDirs := s.getChangedDirs(ctx, allDirs, allDBDirs, lastModifiedSince)
	deletedDirs := s.getDeletedDirs(ctx, allDirs, allDBDirs)

	if len(changedDirs)+len(deletedDirs) == 0 {
		log.Debug(ctx, "No changes found in Music Folder", "folder", s.rootFolder)
		return nil
	}

	if log.CurrentLevel() >= log.LevelTrace {
		log.Info(ctx, "Folder changes detected", "changedFolders", len(changedDirs), "deletedFolders", len(deletedDirs),
			"changed", strings.Join(changedDirs, ";"), "deleted", strings.Join(deletedDirs, ";"))
	} else {
		log.Info(ctx, "Folder changes detected", "changedFolders", len(changedDirs), "deletedFolders", len(deletedDirs))
	}

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

	// Now that all mediafiles are imported/updated, search for and import playlists
	u, _ := request.UserFrom(ctx)
	plsCount := 0
	for _, dir := range changedDirs {
		info := allDirs[dir]
		if info.hasPlaylist {
			if !u.IsAdmin {
				log.Warn("Playlists will not be imported, as there are no admin users yet, "+
					"Please create an admin user first, and then update the playlists for them to be imported", "dir", dir)
			} else {
				plsCount = s.plsSync.processPlaylists(ctx, dir)
			}
		}
	}

	err = s.ds.GC(log.NewContext(ctx))
	log.Info("Finished processing Music Folder", "folder", s.rootFolder, "elapsed", time.Since(start),
		"added", s.cnt.added, "updated", s.cnt.updated, "deleted", s.cnt.deleted, "playlistsImported", plsCount)

	return err
}

func (s *TagScanner2) getDirTree(ctx context.Context) (dirMap, error) {
	start := time.Now()
	log.Trace(ctx, "Loading directory tree from music folder", "folder", s.rootFolder)
	dirs, err := loadDirTree(ctx, s.rootFolder)
	if err != nil {
		return nil, err
	}
	log.Debug("Directory tree loaded from music folder", "total", len(dirs), "elapsed", time.Since(start))
	return dirs, nil
}

func (s *TagScanner2) getDBDirTree(ctx context.Context) (map[string]struct{}, error) {
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

func (s *TagScanner2) getChangedDirs(ctx context.Context, dirs dirMap, dbDirs map[string]struct{}, lastModified time.Time) []string {
	start := time.Now()
	log.Trace(ctx, "Checking for changed folders")
	var changed []string

	for d, info := range dirs {
		_, inDB := dbDirs[d]
		if !inDB && (info.hasAudioFiles) {
			changed = append(changed, d)
			continue
		}
		if info.modTime.After(lastModified) {
			changed = append(changed, d)
		}
	}

	sort.Strings(changed)
	log.Debug(ctx, "Finished changed folders check", "total", len(changed), "elapsed", time.Since(start))
	return changed
}

func (s *TagScanner2) getDeletedDirs(ctx context.Context, allDirs dirMap, dbDirs map[string]struct{}) []string {
	start := time.Now()
	log.Trace(ctx, "Checking for deleted folders")
	var deleted []string

	for d := range dbDirs {
		if _, ok := allDirs[d]; !ok {
			deleted = append(deleted, d)
		}
	}

	sort.Strings(deleted)
	log.Debug(ctx, "Finished deleted folders check", "total", len(deleted), "elapsed", time.Since(start))
	return deleted
}

func (s *TagScanner2) processDeletedDir(ctx context.Context, dir string) error {
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
	}

	err = buffer.flush()
	log.Info(ctx, "Finished processing deleted folder", "path", dir, "purged", len(mfs), "elapsed", time.Since(start))
	return err
}

func (s *TagScanner2) processChangedDir(ctx context.Context, dir string) error {
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
	files, err := LoadAllAudioFiles(dir)
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

	err = buffer.flush()
	log.Info(ctx, "Finished processing changed folder", "dir", dir, "updated", numUpdatedTracks,
		"purged", numPurgedTracks, "elapsed", time.Since(start))
	return err
}

func (s *TagScanner2) deleteOrphanSongs(ctx context.Context, dir string, tracksToDelete map[string]model.MediaFile, buffer *refreshBuffer) (int, error) {
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

func (s *TagScanner2) addOrUpdateTracksInDB(ctx context.Context, dir string, currentTracks map[string]model.MediaFile, filesToUpdate []string, buffer *refreshBuffer) (int, error) {
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

func (s *TagScanner2) withAdminUser(ctx context.Context) context.Context {
	u, err := s.ds.User(ctx).FindFirstAdmin()
	if err != nil {
		log.Warn(ctx, "No admin user found!", err)
		u = &model.User{}
	}

	ctx = request.WithUsername(ctx, u.UserName)
	return request.WithUser(ctx, *u)
}

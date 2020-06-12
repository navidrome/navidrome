package scanner

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/utils"
	"github.com/kennygrant/sanitize"
)

type TagScanner struct {
	rootFolder string
	ds         model.DataStore
	detector   *ChangeDetector
	firstRun   sync.Once
}

func NewTagScanner(rootFolder string, ds model.DataStore) *TagScanner {
	return &TagScanner{
		rootFolder: rootFolder,
		ds:         ds,
		detector:   NewChangeDetector(rootFolder),
		firstRun:   sync.Once{},
	}
}

type (
	ArtistMap map[string]struct{}
	AlbumMap  map[string]struct{}
)

const (
	// batchSize used for albums/artists updates
	batchSize = 5

	// filesBatchSize used for extract file metadata
	filesBatchSize = 100
)

// Scan algorithm overview:
// For each changed folder: Get all files from DB that starts with the folder, scan each file:
//	    if file in folder is newer, update the one in DB
//      if file in folder does not exists in DB, add
// 	    for each file in the DB that is not found in the folder, delete from DB
// For each deleted folder: delete all files from DB that starts with the folder path
// Only on first run, check if any folder under each changed folder is missing.
//      if it is, delete everything under it
// Create new albums/artists, update counters:
//      collect all albumIDs and artistIDs from previous steps
//	    refresh the collected albums and artists with the metadata from the mediafiles
// Delete all empty albums, delete all empty Artists
func (s *TagScanner) Scan(ctx context.Context, lastModifiedSince time.Time) error {
	start := time.Now()
	changed, deleted, err := s.detector.Scan(lastModifiedSince)
	if err != nil {
		return err
	}

	if len(changed)+len(deleted) == 0 {
		log.Debug(ctx, "No changes found in Music Folder", "folder", s.rootFolder)
		return nil
	}

	if log.CurrentLevel() >= log.LevelTrace {
		log.Info(ctx, "Folder changes found", "numChanged", len(changed), "numDeleted", len(deleted),
			"changed", strings.Join(changed, ";"), "deleted", strings.Join(deleted, ";"))
	} else {
		log.Info(ctx, "Folder changes found", "numChanged", len(changed), "numDeleted", len(deleted))
	}

	sort.Strings(changed)
	sort.Strings(deleted)

	updatedArtists := ArtistMap{}
	updatedAlbums := AlbumMap{}

	for _, c := range changed {
		err := s.processChangedDir(ctx, c, updatedArtists, updatedAlbums)
		if err != nil {
			return err
		}
	}
	for _, c := range deleted {
		err := s.processDeletedDir(ctx, c, updatedArtists, updatedAlbums)
		if err != nil {
			return err
		}
	}

	err = s.flushAlbums(ctx, updatedAlbums)
	if err != nil {
		return err
	}

	err = s.flushArtists(ctx, updatedArtists)
	if err != nil {
		return err
	}

	s.firstRun.Do(func() {
		s.removeDeletedFolders(context.TODO(), changed)
	})

	err = s.ds.GC(log.NewContext(context.TODO()))
	log.Info("Finished Music Folder", "folder", s.rootFolder, "elapsed", time.Since(start))

	return err
}

func (s *TagScanner) flushAlbums(ctx context.Context, updatedAlbums AlbumMap) error {
	if len(updatedAlbums) == 0 {
		return nil
	}
	var ids []string
	for id := range updatedAlbums {
		ids = append(ids, id)
		delete(updatedAlbums, id)
	}
	return s.ds.Album(ctx).Refresh(ids...)
}

func (s *TagScanner) flushArtists(ctx context.Context, updatedArtists ArtistMap) error {
	if len(updatedArtists) == 0 {
		return nil
	}
	var ids []string
	for id := range updatedArtists {
		ids = append(ids, id)
		delete(updatedArtists, id)
	}
	return s.ds.Artist(ctx).Refresh(ids...)
}

func (s *TagScanner) processChangedDir(ctx context.Context, dir string, updatedArtists ArtistMap, updatedAlbums AlbumMap) error {
	dir = filepath.Join(s.rootFolder, dir)
	start := time.Now()

	// Load folder's current tracks from DB into a map
	currentTracks := map[string]model.MediaFile{}
	ct, err := s.ds.MediaFile(ctx).FindByPath(dir)
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
	log.Trace("Processing changed folder", "dir", dir, "tracksInDB", len(currentTracks), "tracksInFolder", len(files))
	var filesToUpdate []string
	for filePath, info := range files {
		c, ok := currentTracks[filePath]
		if !ok || (ok && info.ModTime().After(c.UpdatedAt)) {
			filesToUpdate = append(filesToUpdate, filePath)
		}
		delete(currentTracks, filePath)
	}

	numUpdatedTracks := 0
	numPurgedTracks := 0

	if len(filesToUpdate) > 0 {
		// Break the file list in chunks to avoid calling ffmpeg with too many parameters
		chunks := utils.BreakUpStringSlice(filesToUpdate, filesBatchSize)
		for _, chunk := range chunks {
			// Load tracks Metadata from the folder
			newTracks, err := s.loadTracks(chunk)
			if err != nil {
				return err
			}

			// If track from folder is newer than the one in DB, update/insert in DB
			log.Trace("Updating mediaFiles in DB", "dir", dir, "files", chunk, "numFiles", len(chunk))
			for i := range newTracks {
				n := newTracks[i]
				err := s.ds.MediaFile(ctx).Put(&n)
				if err != nil {
					return err
				}
				err = s.updateAlbum(ctx, n.AlbumID, updatedAlbums)
				if err != nil {
					return err
				}
				err = s.updateArtist(ctx, n.AlbumArtistID, updatedArtists)
				if err != nil {
					return err
				}
				numUpdatedTracks++
			}
		}
	}

	if len(currentTracks) > 0 {
		log.Trace("Deleting dangling tracks from DB", "dir", dir, "numTracks", len(currentTracks))
		// Remaining tracks from DB that are not in the folder are deleted
		for _, ct := range currentTracks {
			numPurgedTracks++
			err = s.updateAlbum(ctx, ct.AlbumID, updatedAlbums)
			if err != nil {
				return err
			}
			err = s.updateArtist(ctx, ct.AlbumArtistID, updatedArtists)
			if err != nil {
				return err
			}
			if err := s.ds.MediaFile(ctx).Delete(ct.ID); err != nil {
				return err
			}
		}
	}

	log.Info("Finished processing changed folder", "dir", dir, "updated", numUpdatedTracks, "purged", numPurgedTracks, "elapsed", time.Since(start))
	return nil
}

func (s *TagScanner) updateAlbum(ctx context.Context, albumId string, updatedAlbums AlbumMap) error {
	updatedAlbums[albumId] = struct{}{}
	if len(updatedAlbums) >= batchSize {
		err := s.flushAlbums(ctx, updatedAlbums)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *TagScanner) updateArtist(ctx context.Context, artistId string, updatedArtists ArtistMap) error {
	updatedArtists[artistId] = struct{}{}
	if len(updatedArtists) >= batchSize {
		err := s.flushArtists(ctx, updatedArtists)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *TagScanner) processDeletedDir(ctx context.Context, dir string, updatedArtists ArtistMap, updatedAlbums AlbumMap) error {
	dir = filepath.Join(s.rootFolder, dir)
	start := time.Now()

	ct, err := s.ds.MediaFile(ctx).FindByPath(dir)
	if err != nil {
		return err
	}
	for _, t := range ct {
		err = s.updateAlbum(ctx, t.AlbumID, updatedAlbums)
		if err != nil {
			return err
		}
		err = s.updateArtist(ctx, t.AlbumArtistID, updatedArtists)
		if err != nil {
			return err
		}
	}

	log.Info("Finished processing deleted folder", "dir", dir, "purged", len(ct), "elapsed", time.Since(start))
	return s.ds.MediaFile(ctx).DeleteByPath(dir)
}

func (s *TagScanner) removeDeletedFolders(ctx context.Context, changed []string) {
	for _, dir := range changed {
		fullPath := filepath.Join(s.rootFolder, dir)
		paths, err := s.ds.MediaFile(ctx).FindPathsRecursively(fullPath)
		if err != nil {
			log.Error(ctx, "Error reading paths from DB", "path", dir, err)
			return
		}

		// If a path is unreadable, remove from the DB
		for _, path := range paths {
			if readable, err := utils.IsDirReadable(path); !readable {
				log.Warn(ctx, "Path unavailable. Removing tracks from DB", "path", path, err)
				err = s.ds.MediaFile(ctx).DeleteByPath(path)
				if err != nil {
					log.Error(ctx, "Error removing MediaFiles from DB", "path", path, err)
				}
			}
		}
	}
}

func (s *TagScanner) loadTracks(filePaths []string) (model.MediaFiles, error) {
	mds, err := ExtractAllMetadata(filePaths)
	if err != nil {
		return nil, err
	}

	var mfs model.MediaFiles
	for _, md := range mds {
		mf := s.toMediaFile(md)
		mfs = append(mfs, mf)
	}
	return mfs, nil
}

func (s *TagScanner) toMediaFile(md *Metadata) model.MediaFile {
	mf := &model.MediaFile{}
	mf.ID = s.trackID(md)
	mf.Title = s.mapTrackTitle(md)
	mf.Album = md.Album()
	mf.AlbumID = s.albumID(md)
	mf.Album = s.mapAlbumName(md)
	mf.ArtistID = s.artistID(md)
	mf.Artist = s.mapArtistName(md)
	mf.AlbumArtistID = s.albumArtistID(md)
	mf.AlbumArtist = s.mapAlbumArtistName(md)
	mf.Genre = md.Genre()
	mf.Compilation = md.Compilation()
	mf.Year = md.Year()
	mf.TrackNumber, _ = md.TrackNumber()
	mf.DiscNumber, _ = md.DiscNumber()
	mf.DiscSubtitle = md.DiscSubtitle()
	mf.Duration = md.Duration()
	mf.BitRate = md.BitRate()
	mf.Path = md.FilePath()
	mf.Suffix = md.Suffix()
	mf.Size = md.Size()
	mf.HasCoverArt = md.HasPicture()
	mf.SortTitle = md.SortTitle()
	mf.SortAlbumName = md.SortAlbum()
	mf.SortArtistName = md.SortArtist()
	mf.SortAlbumArtistName = md.SortAlbumArtist()
	mf.OrderAlbumName = sanitizeFieldForSorting(mf.Album)
	mf.OrderArtistName = sanitizeFieldForSorting(mf.Artist)
	mf.OrderAlbumArtistName = sanitizeFieldForSorting(mf.AlbumArtist)

	// TODO Get Creation time. https://github.com/djherbis/times ?
	mf.CreatedAt = md.ModificationTime()
	mf.UpdatedAt = md.ModificationTime()

	return *mf
}

func sanitizeFieldForSorting(originalValue string) string {
	v := utils.NoArticle(originalValue)
	v = strings.TrimSpace(sanitize.Accents(v))
	return utils.NoArticle(v)
}

func (s *TagScanner) mapTrackTitle(md *Metadata) string {
	if md.Title() == "" {
		s := strings.TrimPrefix(md.FilePath(), s.rootFolder+string(os.PathSeparator))
		e := filepath.Ext(s)
		return strings.TrimSuffix(s, e)
	}
	return md.Title()
}

func (s *TagScanner) mapAlbumArtistName(md *Metadata) string {
	switch {
	case md.Compilation():
		return consts.VariousArtists
	case md.AlbumArtist() != "":
		return md.AlbumArtist()
	case md.Artist() != "":
		return md.Artist()
	default:
		return consts.UnknownArtist
	}
}

func (s *TagScanner) mapArtistName(md *Metadata) string {
	if md.Artist() != "" {
		return md.Artist()
	}
	return consts.UnknownArtist
}

func (s *TagScanner) mapAlbumName(md *Metadata) string {
	name := md.Album()
	if name == "" {
		return "[Unknown Album]"
	}
	return name
}

func (s *TagScanner) trackID(md *Metadata) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(md.FilePath())))
}

func (s *TagScanner) albumID(md *Metadata) string {
	albumPath := strings.ToLower(fmt.Sprintf("%s\\%s", s.mapAlbumArtistName(md), s.mapAlbumName(md)))
	return fmt.Sprintf("%x", md5.Sum([]byte(albumPath)))
}

func (s *TagScanner) artistID(md *Metadata) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(s.mapArtistName(md)))))
}

func (s *TagScanner) albumArtistID(md *Metadata) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(s.mapAlbumArtistName(md)))))
}

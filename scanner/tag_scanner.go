package scanner

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
)

type TagScanner struct {
	rootFolder string
	ds         model.DataStore
	detector   *ChangeDetector
}

func NewTagScanner(rootFolder string, ds model.DataStore) *TagScanner {
	return &TagScanner{
		rootFolder: rootFolder,
		ds:         ds,
		detector:   NewChangeDetector(rootFolder),
	}
}

// Scan algorithm overview:
// For each changed: Get all files from DB that starts with the folder, scan each file:
//	    if file in folder is newer, update the one in DB
//      if file in folder does not exists in DB, add
// 	    for each file in the DB that is not found in the folder, delete from DB
// For each deleted folder: delete all files from DB that starts with the folder path
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

	updatedArtists := map[string]bool{}
	updatedAlbums := map[string]bool{}

	for _, c := range changed {
		err := s.processChangedDir(ctx, c, updatedArtists, updatedAlbums)
		if err != nil {
			return err
		}
		if len(updatedAlbums)+len(updatedArtists) > 100 {
			err = s.refreshAlbums(ctx, updatedAlbums)
			if err != nil {
				return err
			}
			err = s.refreshArtists(ctx, updatedArtists)
			if err != nil {
				return err
			}
			updatedAlbums = map[string]bool{}
			updatedArtists = map[string]bool{}
		}
	}
	for _, c := range deleted {
		err := s.processDeletedDir(ctx, c, updatedArtists, updatedAlbums)
		if err != nil {
			return err
		}
		if len(updatedAlbums)+len(updatedArtists) > 100 {
			err = s.refreshAlbums(ctx, updatedAlbums)
			if err != nil {
				return err
			}
			err = s.refreshArtists(ctx, updatedArtists)
			if err != nil {
				return err
			}
			updatedAlbums = map[string]bool{}
			updatedArtists = map[string]bool{}
		}
	}

	err = s.refreshAlbums(ctx, updatedAlbums)
	if err != nil {
		return err
	}

	err = s.refreshArtists(ctx, updatedArtists)
	if err != nil {
		return err
	}

	err = s.ds.GC(log.NewContext(nil))
	log.Info("Finished Music Folder", "folder", s.rootFolder, "elapsed", time.Since(start))

	return err
}

func (s *TagScanner) refreshAlbums(ctx context.Context, updatedAlbums map[string]bool) error {
	var ids []string
	for id := range updatedAlbums {
		ids = append(ids, id)
	}
	return s.ds.Album(ctx).Refresh(ids...)
}

func (s *TagScanner) refreshArtists(ctx context.Context, updatedArtists map[string]bool) error {
	var ids []string
	for id := range updatedArtists {
		ids = append(ids, id)
	}
	return s.ds.Artist(ctx).Refresh(ids...)
}

func (s *TagScanner) processChangedDir(ctx context.Context, dir string, updatedArtists map[string]bool, updatedAlbums map[string]bool) error {
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

	// Load tracks Metadata from the folder
	newTracks, err := s.loadTracks(filesToUpdate)
	if err != nil {
		return err
	}

	// If track from folder is newer than the one in DB, update/insert in DB and delete from the current tracks
	log.Trace("Updating mediaFiles in DB", "dir", dir, "files", filesToUpdate, "numFiles", len(filesToUpdate))
	numUpdatedTracks := 0
	numPurgedTracks := 0
	for _, n := range newTracks {
		err := s.ds.MediaFile(ctx).Put(&n)
		updatedArtists[n.ArtistID] = true
		updatedAlbums[n.AlbumID] = true
		numUpdatedTracks++
		if err != nil {
			return err
		}
	}

	// Remaining tracks from DB that are not in the folder are deleted
	for _, ct := range currentTracks {
		numPurgedTracks++
		updatedArtists[ct.ArtistID] = true
		updatedAlbums[ct.AlbumID] = true
		if err := s.ds.MediaFile(ctx).Delete(ct.ID); err != nil {
			return err
		}
	}

	log.Info("Finished processing changed folder", "dir", dir, "updated", numUpdatedTracks, "purged", numPurgedTracks, "elapsed", time.Since(start))
	return nil
}

func (s *TagScanner) processDeletedDir(ctx context.Context, dir string, updatedArtists map[string]bool, updatedAlbums map[string]bool) error {
	dir = filepath.Join(s.rootFolder, dir)
	start := time.Now()

	ct, err := s.ds.MediaFile(ctx).FindByPath(dir)
	if err != nil {
		return err
	}
	for _, t := range ct {
		updatedArtists[t.ArtistID] = true
		updatedAlbums[t.AlbumID] = true
	}

	log.Info("Finished processing deleted folder", "dir", dir, "deleted", len(ct), "elapsed", time.Since(start))
	return s.ds.MediaFile(ctx).DeleteByPath(dir)
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
	mf := model.MediaFile{}
	mf.ID = s.trackID(md)
	mf.Title = s.mapTrackTitle(md)
	mf.Album = md.Album()
	mf.AlbumID = s.albumID(md)
	mf.Album = s.mapAlbumName(md)
	if md.Artist() == "" {
		mf.Artist = consts.UnknownArtist
	} else {
		mf.Artist = md.Artist()
	}
	mf.ArtistID = s.artistID(md)
	mf.AlbumArtist = md.AlbumArtist()
	mf.Genre = md.Genre()
	mf.Compilation = md.Compilation()
	mf.Year = md.Year()
	mf.TrackNumber, _ = md.TrackNumber()
	mf.DiscNumber, _ = md.DiscNumber()
	mf.Duration = md.Duration()
	mf.BitRate = md.BitRate()
	mf.Path = md.FilePath()
	mf.Suffix = md.Suffix()
	mf.Size = md.Size()
	mf.HasCoverArt = md.HasPicture()

	// TODO Get Creation time. https://github.com/djherbis/times ?
	mf.CreatedAt = md.ModificationTime()
	mf.UpdatedAt = md.ModificationTime()

	return mf
}

func (s *TagScanner) mapTrackTitle(md *Metadata) string {
	if md.Title() == "" {
		s := strings.TrimPrefix(md.FilePath(), s.rootFolder+string(os.PathSeparator))
		e := filepath.Ext(s)
		return strings.TrimSuffix(s, e)
	}
	return md.Title()
}

func (s *TagScanner) mapArtistName(md *Metadata) string {
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
	albumPath := strings.ToLower(fmt.Sprintf("%s\\%s", s.mapArtistName(md), s.mapAlbumName(md)))
	return fmt.Sprintf("%x", md5.Sum([]byte(albumPath)))
}

func (s *TagScanner) artistID(md *Metadata) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(s.mapArtistName(md)))))
}

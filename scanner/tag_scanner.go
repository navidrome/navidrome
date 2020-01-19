package scanner

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/model"
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
	changed, deleted, err := s.detector.Scan(lastModifiedSince)
	if err != nil {
		return err
	}

	if len(changed)+len(deleted) == 0 {
		return nil
	}

	log.Info("Folder changes found", "changed", len(changed), "deleted", len(deleted))

	sort.Strings(changed)
	sort.Strings(deleted)

	updatedArtists := map[string]bool{}
	updatedAlbums := map[string]bool{}

	for _, c := range changed {
		err := s.processChangedDir(c, updatedArtists, updatedAlbums)
		if err != nil {
			return err
		}
		if len(updatedAlbums)+len(updatedArtists) > 100 {
			err = s.refreshAlbums(updatedAlbums)
			if err != nil {
				return err
			}
			err = s.refreshArtists(updatedArtists)
			if err != nil {
				return err
			}
			updatedAlbums = map[string]bool{}
			updatedArtists = map[string]bool{}
		}
	}
	for _, c := range deleted {
		err := s.processDeletedDir(c, updatedArtists, updatedAlbums)
		if err != nil {
			return err
		}
		if len(updatedAlbums)+len(updatedArtists) > 100 {
			err = s.refreshAlbums(updatedAlbums)
			if err != nil {
				return err
			}
			err = s.refreshArtists(updatedArtists)
			if err != nil {
				return err
			}
			updatedAlbums = map[string]bool{}
			updatedArtists = map[string]bool{}
		}
	}

	err = s.refreshAlbums(updatedAlbums)
	if err != nil {
		return err
	}

	err = s.refreshArtists(updatedArtists)
	if err != nil {
		return err
	}

	err = s.ds.Album().PurgeEmpty()
	if err != nil {
		return err
	}

	err = s.ds.Artist().PurgeEmpty()
	if err != nil {
		return err
	}

	return nil
}

func (s *TagScanner) refreshAlbums(updatedAlbums map[string]bool) error {
	var ids []string
	for id := range updatedAlbums {
		ids = append(ids, id)
	}
	return s.ds.Album().Refresh(ids...)
}

func (s *TagScanner) refreshArtists(updatedArtists map[string]bool) error {
	var ids []string
	for id := range updatedArtists {
		ids = append(ids, id)
	}
	return s.ds.Artist().Refresh(ids...)
}

func (s *TagScanner) processChangedDir(dir string, updatedArtists map[string]bool, updatedAlbums map[string]bool) error {
	dir = path.Join(s.rootFolder, dir)

	start := time.Now()

	// Load folder's current tracks from DB into a map
	currentTracks := map[string]model.MediaFile{}
	ct, err := s.ds.MediaFile().FindByPath(dir)
	if err != nil {
		return err
	}
	for _, t := range ct {
		currentTracks[t.ID] = t
		updatedArtists[t.ArtistID] = true
		updatedAlbums[t.AlbumID] = true
	}

	// Load tracks from the folder
	newTracks, err := s.loadTracks(dir)
	if err != nil {
		return err
	}

	// If no tracks to process, return
	if len(newTracks)+len(currentTracks) == 0 {
		return nil
	}

	// If track from folder is newer than the one in DB, update/insert in DB and delete from the current tracks
	log.Trace("Processing changed folder", "dir", dir, "tracksInDB", len(currentTracks), "tracksInFolder", len(newTracks))
	numUpdatedTracks := 0
	numPurgedTracks := 0
	for _, n := range newTracks {
		c, ok := currentTracks[n.ID]
		if !ok || (ok && n.UpdatedAt.After(c.UpdatedAt)) {
			err := s.ds.MediaFile().Put(&n, false)
			updatedArtists[n.ArtistID] = true
			updatedAlbums[n.AlbumID] = true
			numUpdatedTracks++
			if err != nil {
				return err
			}
		}
		delete(currentTracks, n.ID)
	}

	// Remaining tracks from DB that are not in the folder are deleted
	for id := range currentTracks {
		numPurgedTracks++
		if err := s.ds.MediaFile().Delete(id); err != nil {
			return err
		}
	}

	log.Debug("Finished processing changed folder", "dir", dir, "updated", numUpdatedTracks, "purged", numPurgedTracks, "elapsed", time.Since(start))
	return nil
}

func (s *TagScanner) processDeletedDir(dir string, updatedArtists map[string]bool, updatedAlbums map[string]bool) error {
	dir = path.Join(s.rootFolder, dir)

	ct, err := s.ds.MediaFile().FindByPath(dir)
	if err != nil {
		return err
	}
	for _, t := range ct {
		updatedArtists[t.ArtistID] = true
		updatedAlbums[t.AlbumID] = true
	}

	return s.ds.MediaFile().DeleteByPath(dir)
}

func (s *TagScanner) loadTracks(dirPath string) (model.MediaFiles, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}
	files, err := dir.Readdir(-1)
	if err != nil {
		return nil, err
	}
	var mds model.MediaFiles
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		filePath := path.Join(dirPath, f.Name())
		md, err := ExtractMetadata(filePath)
		if err != nil {
			continue
		}
		mf := s.toMediaFile(md)
		mds = append(mds, mf)
	}
	return mds, nil
}

func (s *TagScanner) toMediaFile(md *Metadata) model.MediaFile {
	mf := model.MediaFile{}
	mf.ID = s.trackID(md)
	mf.Title = s.mapTrackTitle(md)
	mf.Album = md.Album()
	mf.AlbumID = s.albumID(md)
	mf.Album = s.mapAlbumName(md)
	if md.Artist() == "" {
		mf.Artist = "[Unknown Artist]"
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
	mf.Size = strconv.Itoa(md.Size())
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
		return "Various Artists"
	case md.AlbumArtist() != "":
		return md.AlbumArtist()
	case md.Artist() != "":
		return md.Artist()
	default:
		return "[Unknown Artist]"
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

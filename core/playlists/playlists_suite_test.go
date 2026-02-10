package playlists_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlaylists(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Playlists Suite")
}

// mockedMediaFileRepo's FindByPaths method returns MediaFiles for the given paths.
// If data map is provided, looks up files by key; otherwise creates them from paths.
type mockedMediaFileRepo struct {
	model.MediaFileRepository
	data map[string]model.MediaFile
}

func (r *mockedMediaFileRepo) FindByPaths(paths []string) (model.MediaFiles, error) {
	var mfs model.MediaFiles

	// If data map provided, look up files
	if r.data != nil {
		for _, path := range paths {
			if mf, ok := r.data[path]; ok {
				mfs = append(mfs, mf)
			}
		}
		return mfs, nil
	}

	// Otherwise, create MediaFiles from paths
	for idx, path := range paths {
		// Strip library qualifier if present (format: "libraryID:path")
		actualPath := path
		libraryID := 1
		if parts := strings.SplitN(path, ":", 2); len(parts) == 2 {
			if id, err := strconv.Atoi(parts[0]); err == nil {
				libraryID = id
				actualPath = parts[1]
			}
		}

		mfs = append(mfs, model.MediaFile{
			ID:        strconv.Itoa(idx),
			Path:      actualPath,
			LibraryID: libraryID,
		})
	}
	return mfs, nil
}

// mockedMediaFileFromListRepo's FindByPaths method returns a list of MediaFiles based on the data field
type mockedMediaFileFromListRepo struct {
	model.MediaFileRepository
	data []string
}

func (r *mockedMediaFileFromListRepo) FindByPaths(paths []string) (model.MediaFiles, error) {
	var mfs model.MediaFiles

	for idx, dataPath := range r.data {
		for _, requestPath := range paths {
			// Strip library qualifier if present (format: "libraryID:path")
			actualPath := requestPath
			libraryID := 1
			if parts := strings.SplitN(requestPath, ":", 2); len(parts) == 2 {
				if id, err := strconv.Atoi(parts[0]); err == nil {
					libraryID = id
					actualPath = parts[1]
				}
			}

			// Case-insensitive comparison (like SQL's "collate nocase"), but with no
			// implicit Unicode normalization (SQLite does not normalize NFC/NFD).
			if strings.EqualFold(actualPath, dataPath) {
				mfs = append(mfs, model.MediaFile{
					ID:        strconv.Itoa(idx),
					Path:      dataPath, // Return original path from DB
					LibraryID: libraryID,
				})
				break
			}
		}
	}
	return mfs, nil
}

type mockedPlaylistRepo struct {
	model.PlaylistRepository
	last     *model.Playlist
	data     map[string]*model.Playlist // keyed by path
	entities map[string]*model.Playlist // keyed by ID
	deleted  []string
	tracks   *mockedPlaylistTrackRepo
}

func (r *mockedPlaylistRepo) FindByPath(path string) (*model.Playlist, error) {
	if r.data != nil {
		if pls, ok := r.data[path]; ok {
			return pls, nil
		}
	}
	return nil, model.ErrNotFound
}

func (r *mockedPlaylistRepo) Put(pls *model.Playlist) error {
	if pls.ID == "" {
		pls.ID = "new-id"
	}
	r.last = pls
	if r.entities != nil {
		r.entities[pls.ID] = pls
	}
	return nil
}

func (r *mockedPlaylistRepo) Get(id string) (*model.Playlist, error) {
	if r.entities != nil {
		if pls, ok := r.entities[id]; ok {
			return pls, nil
		}
	}
	return nil, model.ErrNotFound
}

func (r *mockedPlaylistRepo) GetWithTracks(id string, _, _ bool) (*model.Playlist, error) {
	return r.Get(id)
}

func (r *mockedPlaylistRepo) Delete(id string) error {
	r.deleted = append(r.deleted, id)
	return nil
}

func (r *mockedPlaylistRepo) Tracks(_ string, _ bool) model.PlaylistTrackRepository {
	return r.tracks
}

type mockedPlaylistTrackRepo struct {
	model.PlaylistTrackRepository
	addedIds   []string
	deletedIds []string
	reordered  bool
	addCount   int
	err        error
}

func (r *mockedPlaylistTrackRepo) Add(ids []string) (int, error) {
	r.addedIds = append(r.addedIds, ids...)
	if r.err != nil {
		return 0, r.err
	}
	return r.addCount, nil
}

func (r *mockedPlaylistTrackRepo) AddAlbums(_ []string) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	return r.addCount, nil
}

func (r *mockedPlaylistTrackRepo) AddArtists(_ []string) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	return r.addCount, nil
}

func (r *mockedPlaylistTrackRepo) AddDiscs(_ []model.DiscID) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	return r.addCount, nil
}

func (r *mockedPlaylistTrackRepo) Delete(ids ...string) error {
	r.deletedIds = append(r.deletedIds, ids...)
	return r.err
}

func (r *mockedPlaylistTrackRepo) Reorder(_, _ int) error {
	r.reordered = true
	return r.err
}

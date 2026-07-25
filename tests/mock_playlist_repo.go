package tests

import (
	"errors"
	"time"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

func CreateMockPlaylistRepo() *MockPlaylistRepo {
	return &MockPlaylistRepo{
		Data:    make(map[string]*model.Playlist),
		PathMap: make(map[string]*model.Playlist),
	}
}

type MockPlaylistRepo struct {
	model.PlaylistRepository
	Data       map[string]*model.Playlist // keyed by ID
	PathMap    map[string]*model.Playlist // keyed by path
	All        model.Playlists
	Options    model.QueryOptions
	Last       *model.Playlist
	Deleted    []string
	Starred    map[string]bool // itemID -> starred
	Ratings    map[string]int  // itemID -> rating
	Err        bool
	TracksRepo model.PlaylistTrackRepository
}

func (m *MockPlaylistRepo) SetError(err bool) {
	m.Err = err
}

func (m *MockPlaylistRepo) SetData(playlists model.Playlists) {
	m.Data = make(map[string]*model.Playlist, len(playlists))
	m.All = playlists
	for i, p := range m.All {
		m.Data[p.ID] = &m.All[i]
	}
}

func (m *MockPlaylistRepo) GetAll(options ...model.QueryOptions) (model.Playlists, error) {
	if len(options) > 0 {
		m.Options = options[0]
	}
	if m.Err {
		return nil, errors.New("error")
	}
	return m.All, nil
}

func (m *MockPlaylistRepo) GetCursor(options ...model.QueryOptions) (model.PlaylistCursor, error) {
	res, err := m.GetAll(options...)
	if err != nil {
		return nil, err
	}
	return func(yield func(model.Playlist, error) bool) {
		for _, p := range res {
			if !yield(p, nil) {
				return
			}
		}
	}, nil
}

func (m *MockPlaylistRepo) Get(id string) (*model.Playlist, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	if m.Data != nil {
		if pls, ok := m.Data[id]; ok {
			return pls, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *MockPlaylistRepo) GetWithTracks(id string, _, _ bool) (*model.Playlist, error) {
	return m.Get(id)
}

func (m *MockPlaylistRepo) Put(pls *model.Playlist, _ ...string) error {
	if m.Err {
		return errors.New("error")
	}
	if pls.ID == "" {
		pls.ID = id.NewRandom()
	}
	m.Last = pls
	if m.Data != nil {
		m.Data[pls.ID] = pls
	}
	return nil
}

func (m *MockPlaylistRepo) FindByPath(path string) (*model.Playlist, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	if m.PathMap != nil {
		if pls, ok := m.PathMap[path]; ok {
			return pls, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *MockPlaylistRepo) Delete(id string) error {
	if m.Err {
		return errors.New("error")
	}
	m.Deleted = append(m.Deleted, id)
	return nil
}

func (m *MockPlaylistRepo) SetStar(starred bool, ids ...string) error {
	if m.Err {
		return errors.New("error")
	}
	if m.Starred == nil {
		m.Starred = map[string]bool{}
	}
	for _, id := range ids {
		m.Starred[id] = starred
	}
	return nil
}

func (m *MockPlaylistRepo) SetRating(rating int, id string) error {
	if m.Err {
		return errors.New("error")
	}
	if m.Ratings == nil {
		m.Ratings = map[string]int{}
	}
	m.Ratings[id] = rating
	return nil
}

func (m *MockPlaylistRepo) IncPlayCount(string, time.Time) error {
	if m.Err {
		return errors.New("error")
	}
	return nil
}

func (m *MockPlaylistRepo) ReassignAnnotation(string, string) error {
	if m.Err {
		return errors.New("error")
	}
	return nil
}

func (m *MockPlaylistRepo) Tracks(_ string, _ bool) model.PlaylistTrackRepository {
	return m.TracksRepo
}

func (m *MockPlaylistRepo) Exists(id string) (bool, error) {
	if m.Err {
		return false, errors.New("error")
	}
	if m.Data != nil {
		_, found := m.Data[id]
		return found, nil
	}
	return false, nil
}

func (m *MockPlaylistRepo) Count(_ ...rest.QueryOptions) (int64, error) {
	if m.Err {
		return 0, errors.New("error")
	}
	return int64(len(m.Data)), nil
}

func (m *MockPlaylistRepo) CountAll(_ ...model.QueryOptions) (int64, error) {
	if m.Err {
		return 0, errors.New("error")
	}
	return int64(len(m.Data)), nil
}

var _ model.PlaylistRepository = (*MockPlaylistRepo)(nil)

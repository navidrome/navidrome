package tests

import (
	"context"
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
	Data            map[string]*model.Playlist // keyed by ID
	PathMap         map[string]*model.Playlist // keyed by path
	All             model.Playlists
	Options         model.QueryOptions
	Last            *model.Playlist
	Deleted         []string
	Starred         map[string]bool // itemID -> starred
	Ratings         map[string]int  // itemID -> rating
	Err             bool
	TracksRepo      model.PlaylistTrackRepository
	TracksRefreshed bool
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

func (m *MockPlaylistRepo) GetAll(_ context.Context, options ...model.QueryOptions) (model.Playlists, error) {
	if len(options) > 0 {
		m.Options = options[0]
	}
	if m.Err {
		return nil, errors.New("error")
	}
	return m.All, nil
}

func (m *MockPlaylistRepo) GetCursor(ctx context.Context, options ...model.QueryOptions) (model.PlaylistCursor, error) {
	res, err := m.GetAll(ctx, options...)
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

func (m *MockPlaylistRepo) Get(_ context.Context, id string) (*model.Playlist, error) {
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

func (m *MockPlaylistRepo) GetWithTracks(ctx context.Context, id string, _, _ bool) (*model.Playlist, error) {
	return m.Get(ctx, id)
}

func (m *MockPlaylistRepo) Put(_ context.Context, pls *model.Playlist, _ ...string) error {
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

func (m *MockPlaylistRepo) FindByPath(_ context.Context, path string) (*model.Playlist, error) {
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

func (m *MockPlaylistRepo) Delete(_ context.Context, ids ...string) error {
	if m.Err {
		return errors.New("error")
	}
	m.Deleted = append(m.Deleted, ids...)
	return nil
}

func (m *MockPlaylistRepo) SetStar(_ context.Context, starred bool, ids ...string) error {
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

func (m *MockPlaylistRepo) SetRating(_ context.Context, rating int, id string) error {
	if m.Err {
		return errors.New("error")
	}
	if m.Ratings == nil {
		m.Ratings = map[string]int{}
	}
	m.Ratings[id] = rating
	return nil
}

func (m *MockPlaylistRepo) IncPlayCount(context.Context, string, time.Time) error {
	if m.Err {
		return errors.New("error")
	}
	return nil
}

func (m *MockPlaylistRepo) ReassignAnnotation(context.Context, string, string) error {
	if m.Err {
		return errors.New("error")
	}
	return nil
}

func (m *MockPlaylistRepo) Tracks(_ context.Context, _ string, refreshSmartPlaylist bool) model.PlaylistTrackRepository {
	m.TracksRefreshed = refreshSmartPlaylist
	return m.TracksRepo
}

func (m *MockPlaylistRepo) Exists(_ context.Context, id string) (bool, error) {
	if m.Err {
		return false, errors.New("error")
	}
	if m.Data != nil {
		_, found := m.Data[id]
		return found, nil
	}
	return false, nil
}

func (m *MockPlaylistRepo) Count(_ context.Context, _ ...rest.QueryOptions) (int64, error) {
	if m.Err {
		return 0, errors.New("error")
	}
	return int64(len(m.Data)), nil
}

func (m *MockPlaylistRepo) CountAll(_ context.Context, _ ...model.QueryOptions) (int64, error) {
	if m.Err {
		return 0, errors.New("error")
	}
	return int64(len(m.Data)), nil
}

var _ model.PlaylistRepository = (*MockPlaylistRepo)(nil)

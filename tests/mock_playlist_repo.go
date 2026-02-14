package tests

import (
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
)

type MockPlaylistRepo struct {
	model.PlaylistRepository

	Entity           *model.Playlist
	Error            error
	SetStarWasCalled bool
}

func (m *MockPlaylistRepo) Get(_ string) (*model.Playlist, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.Entity == nil {
		return nil, model.ErrNotFound
	}
	return m.Entity, nil
}

func (m *MockPlaylistRepo) Count(_ ...rest.QueryOptions) (int64, error) {
	if m.Error != nil {
		return 0, m.Error
	}
	if m.Entity == nil {
		return 0, nil
	}
	return 1, nil
}

func (m *MockPlaylistRepo) Exists(_ string) (bool, error) {
	if m.Error != nil {
		return false, m.Error
	}
	return m.Entity != nil, nil
}

func (m *MockPlaylistRepo) SetStar(starred bool, itemIDs ...string) error {
	if m.Error != nil {
		return m.Error
	}
	m.SetStarWasCalled = true
	return nil
}

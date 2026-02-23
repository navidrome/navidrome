package tests

import "github.com/navidrome/navidrome/model"

type MockPlaylistTrackRepo struct {
	model.PlaylistTrackRepository
	AddedIds   []string
	DeletedIds []string
	Reordered  bool
	AddCount   int
	Err        error
}

func (m *MockPlaylistTrackRepo) Add(ids []string) (int, error) {
	m.AddedIds = append(m.AddedIds, ids...)
	if m.Err != nil {
		return 0, m.Err
	}
	return m.AddCount, nil
}

func (m *MockPlaylistTrackRepo) AddAlbums(_ []string) (int, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	return m.AddCount, nil
}

func (m *MockPlaylistTrackRepo) AddArtists(_ []string) (int, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	return m.AddCount, nil
}

func (m *MockPlaylistTrackRepo) AddDiscs(_ []model.DiscID) (int, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	return m.AddCount, nil
}

func (m *MockPlaylistTrackRepo) Delete(ids ...string) error {
	m.DeletedIds = append(m.DeletedIds, ids...)
	return m.Err
}

func (m *MockPlaylistTrackRepo) Reorder(_, _ int) error {
	m.Reordered = true
	return m.Err
}

var _ model.PlaylistTrackRepository = (*MockPlaylistTrackRepo)(nil)

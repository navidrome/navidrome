package tests

import (
	"errors"

	"github.com/navidrome/navidrome/model"
)

func CreateMockArtistRepo() *MockArtistRepo {
	return &MockArtistRepo{}
}

type MockArtistRepo struct {
	model.ArtistRepository
	data map[string]model.Artist
	err  bool
}

func (m *MockArtistRepo) SetError(err bool) {
	m.err = err
}

func (m *MockArtistRepo) SetData(artists model.Artists) {
	m.data = make(map[string]model.Artist)
	for _, a := range artists {
		m.data[a.ID] = a
	}
}

func (m *MockArtistRepo) Exists(id string) (bool, error) {
	if m.err {
		return false, errors.New("Error!")
	}
	_, found := m.data[id]
	return found, nil
}

func (m *MockArtistRepo) Get(id string) (*model.Artist, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	if d, ok := m.data[id]; ok {
		return &d, nil
	}
	return nil, model.ErrNotFound
}

var _ model.ArtistRepository = (*MockArtistRepo)(nil)

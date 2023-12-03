package tests

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/navidrome/navidrome/model"
)

func CreateMockArtistRepo() *MockArtistRepo {
	return &MockArtistRepo{
		data: make(map[string]*model.Artist),
	}
}

type MockArtistRepo struct {
	model.ArtistRepository
	data map[string]*model.Artist
	err  bool
}

func (m *MockArtistRepo) SetError(err bool) {
	m.err = err
}

func (m *MockArtistRepo) SetData(artists model.Artists) {
	m.data = make(map[string]*model.Artist)
	for i, a := range artists {
		m.data[a.ID] = &artists[i]
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
		return d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockArtistRepo) Put(ar *model.Artist, columsToUpdate ...string) error {
	if m.err {
		return errors.New("error")
	}
	if ar.ID == "" {
		ar.ID = uuid.NewString()
	}
	m.data[ar.ID] = ar
	return nil
}

func (m *MockArtistRepo) IncPlayCount(id string, timestamp time.Time) error {
	if m.err {
		return errors.New("error")
	}
	if d, ok := m.data[id]; ok {
		d.PlayCount++
		d.PlayDate = &timestamp
		return nil
	}
	return model.ErrNotFound
}

var _ model.ArtistRepository = (*MockArtistRepo)(nil)

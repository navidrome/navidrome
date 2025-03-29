package tests

import (
	"errors"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

func CreateMockArtistRepo() *MockArtistRepo {
	return &MockArtistRepo{
		Data: make(map[string]*model.Artist),
	}
}

type MockArtistRepo struct {
	model.ArtistRepository
	Data map[string]*model.Artist
	Err  bool
}

func (m *MockArtistRepo) SetError(err bool) {
	m.Err = err
}

func (m *MockArtistRepo) SetData(artists model.Artists) {
	m.Data = make(map[string]*model.Artist)
	for i, a := range artists {
		m.Data[a.ID] = &artists[i]
	}
}

func (m *MockArtistRepo) Exists(id string) (bool, error) {
	if m.Err {
		return false, errors.New("Error!")
	}
	_, found := m.Data[id]
	return found, nil
}

func (m *MockArtistRepo) Get(id string) (*model.Artist, error) {
	if m.Err {
		return nil, errors.New("Error!")
	}
	if d, ok := m.Data[id]; ok {
		return d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockArtistRepo) Put(ar *model.Artist, columsToUpdate ...string) error {
	if m.Err {
		return errors.New("error")
	}
	if ar.ID == "" {
		ar.ID = id.NewRandom()
	}
	m.Data[ar.ID] = ar
	return nil
}

func (m *MockArtistRepo) IncPlayCount(id string, timestamp time.Time) error {
	if m.Err {
		return errors.New("error")
	}
	if d, ok := m.Data[id]; ok {
		d.PlayCount++
		d.PlayDate = &timestamp
		return nil
	}
	return model.ErrNotFound
}

var _ model.ArtistRepository = (*MockArtistRepo)(nil)

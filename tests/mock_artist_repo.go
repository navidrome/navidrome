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
	Data    map[string]*model.Artist
	Err     bool
	Options model.QueryOptions
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

func (m *MockArtistRepo) GetAll(options ...model.QueryOptions) (model.Artists, error) {
	if len(options) > 0 {
		m.Options = options[0]
	}
	if m.Err {
		return nil, errors.New("mock repo error")
	}
	var allArtists model.Artists
	for _, artist := range m.Data {
		allArtists = append(allArtists, *artist)
	}
	// Apply Max=1 if present (simple simulation for findArtistByName)
	if len(options) > 0 && options[0].Max == 1 && len(allArtists) > 0 {
		return allArtists[:1], nil
	}
	return allArtists, nil
}

func (m *MockArtistRepo) UpdateExternalInfo(artist *model.Artist) error {
	if m.Err {
		return errors.New("mock repo error")
	}
	return nil
}

func (m *MockArtistRepo) RefreshStats(allArtists bool) (int64, error) {
	if m.Err {
		return 0, errors.New("mock repo error")
	}
	return int64(len(m.Data)), nil
}

func (m *MockArtistRepo) RefreshPlayCounts() (int64, error) {
	if m.Err {
		return 0, errors.New("mock repo error")
	}
	return int64(len(m.Data)), nil
}

func (m *MockArtistRepo) GetIndex(includeMissing bool, libraryIds []int, roles ...model.Role) (model.ArtistIndexes, error) {
	if m.Err {
		return nil, errors.New("mock repo error")
	}

	artists, err := m.GetAll()
	if err != nil {
		return nil, err
	}

	// For mock purposes, if no artists available, return empty result
	if len(artists) == 0 {
		return model.ArtistIndexes{}, nil
	}

	// Simple index grouping by first letter (simplified implementation for mocks)
	indexMap := make(map[string]model.Artists)
	for _, artist := range artists {
		key := "#"
		if len(artist.Name) > 0 {
			key = string(artist.Name[0])
		}
		indexMap[key] = append(indexMap[key], artist)
	}

	var result model.ArtistIndexes
	for k, artists := range indexMap {
		result = append(result, model.ArtistIndex{ID: k, Artists: artists})
	}

	return result, nil
}

func (m *MockArtistRepo) Search(q string, offset int, size int, options ...model.QueryOptions) (model.Artists, error) {
	if len(options) > 0 {
		m.Options = options[0]
	}
	if m.Err {
		return nil, errors.New("unexpected error")
	}
	// Simple mock implementation - just return all artists for testing
	allArtists, err := m.GetAll()
	return allArtists, err
}

var _ model.ArtistRepository = (*MockArtistRepo)(nil)

package tests

import (
	"errors"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

func CreateMockAlbumRepo() *MockAlbumRepo {
	return &MockAlbumRepo{
		Data: make(map[string]*model.Album),
	}
}

type MockAlbumRepo struct {
	model.AlbumRepository
	Data                    map[string]*model.Album
	All                     model.Albums
	Err                     bool
	Options                 model.QueryOptions
	ReassignAnnotationCalls map[string]string // prevID -> newID
}

func (m *MockAlbumRepo) SetError(err bool) {
	m.Err = err
}

func (m *MockAlbumRepo) SetData(albums model.Albums) {
	m.Data = make(map[string]*model.Album, len(albums))
	m.All = albums
	for i, a := range m.All {
		m.Data[a.ID] = &m.All[i]
	}
}

func (m *MockAlbumRepo) Exists(id string) (bool, error) {
	if m.Err {
		return false, errors.New("unexpected error")
	}
	_, found := m.Data[id]
	return found, nil
}

func (m *MockAlbumRepo) Get(id string) (*model.Album, error) {
	if m.Err {
		return nil, errors.New("unexpected error")
	}
	if d, ok := m.Data[id]; ok {
		return d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockAlbumRepo) Put(al *model.Album) error {
	if m.Err {
		return errors.New("unexpected error")
	}
	if al.ID == "" {
		al.ID = id.NewRandom()
	}
	m.Data[al.ID] = al
	return nil
}

func (m *MockAlbumRepo) GetAll(qo ...model.QueryOptions) (model.Albums, error) {
	if len(qo) > 0 {
		m.Options = qo[0]
	}
	if m.Err {
		return nil, errors.New("unexpected error")
	}
	return m.All, nil
}

func (m *MockAlbumRepo) IncPlayCount(id string, timestamp time.Time) error {
	if m.Err {
		return errors.New("unexpected error")
	}
	if d, ok := m.Data[id]; ok {
		d.PlayCount++
		d.PlayDate = &timestamp
		return nil
	}
	return model.ErrNotFound
}
func (m *MockAlbumRepo) CountAll(...model.QueryOptions) (int64, error) {
	return int64(len(m.All)), nil
}

func (m *MockAlbumRepo) GetTouchedAlbums(libID int) (model.AlbumCursor, error) {
	if m.Err {
		return nil, errors.New("unexpected error")
	}
	return func(yield func(model.Album, error) bool) {
		for _, a := range m.Data {
			if a.ID == "error" {
				if !yield(*a, errors.New("error")) {
					break
				}
				continue
			}
			if a.LibraryID != libID {
				continue
			}
			if !yield(*a, nil) {
				break
			}
		}
	}, nil
}

func (m *MockAlbumRepo) UpdateExternalInfo(album *model.Album) error {
	if m.Err {
		return errors.New("unexpected error")
	}
	return nil
}

func (m *MockAlbumRepo) Search(q string, offset int, size int, options ...model.QueryOptions) (model.Albums, error) {
	if len(options) > 0 {
		m.Options = options[0]
	}
	if m.Err {
		return nil, errors.New("unexpected error")
	}
	// Simple mock implementation - just return all albums for testing
	return m.All, nil
}

// ReassignAnnotation reassigns annotations from one album to another
func (m *MockAlbumRepo) ReassignAnnotation(prevID string, newID string) error {
	if m.Err {
		return errors.New("unexpected error")
	}
	// Mock implementation - track the reassignment calls
	if m.ReassignAnnotationCalls == nil {
		m.ReassignAnnotationCalls = make(map[string]string)
	}
	m.ReassignAnnotationCalls[prevID] = newID
	return nil
}

// SetRating sets the rating for an album
func (m *MockAlbumRepo) SetRating(rating int, itemID string) error {
	if m.Err {
		return errors.New("unexpected error")
	}
	return nil
}

// SetStar sets the starred status for albums
func (m *MockAlbumRepo) SetStar(starred bool, itemIDs ...string) error {
	if m.Err {
		return errors.New("unexpected error")
	}
	return nil
}

var _ model.AlbumRepository = (*MockAlbumRepo)(nil)

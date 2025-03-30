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
	Data    map[string]*model.Album
	All     model.Albums
	Err     bool
	Options model.QueryOptions
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

var _ model.AlbumRepository = (*MockAlbumRepo)(nil)

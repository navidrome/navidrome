package tests

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/navidrome/navidrome/model"
)

func CreateMockAlbumRepo() *MockAlbumRepo {
	return &MockAlbumRepo{
		data: make(map[string]*model.Album),
	}
}

type MockAlbumRepo struct {
	model.AlbumRepository
	data    map[string]*model.Album
	all     model.Albums
	err     bool
	Options model.QueryOptions
}

func (m *MockAlbumRepo) SetError(err bool) {
	m.err = err
}

func (m *MockAlbumRepo) SetData(albums model.Albums) {
	m.data = make(map[string]*model.Album)
	m.all = albums
	for i, a := range m.all {
		m.data[a.ID] = &m.all[i]
	}
}

func (m *MockAlbumRepo) Exists(id string) (bool, error) {
	if m.err {
		return false, errors.New("Error!")
	}
	_, found := m.data[id]
	return found, nil
}

func (m *MockAlbumRepo) Get(id string) (*model.Album, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	if d, ok := m.data[id]; ok {
		return d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockAlbumRepo) Put(al *model.Album) error {
	if m.err {
		return errors.New("error")
	}
	if al.ID == "" {
		al.ID = uuid.NewString()
	}
	m.data[al.ID] = al
	return nil
}

func (m *MockAlbumRepo) GetAll(qo ...model.QueryOptions) (model.Albums, error) {
	if len(qo) > 0 {
		m.Options = qo[0]
	}
	if m.err {
		return nil, errors.New("Error!")
	}
	return m.all, nil
}

func (m *MockAlbumRepo) GetAllWithoutGenres(qo ...model.QueryOptions) (model.Albums, error) {
	return m.GetAll(qo...)
}

func (m *MockAlbumRepo) IncPlayCount(id string, timestamp time.Time) error {
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
func (m *MockAlbumRepo) CountAll(...model.QueryOptions) (int64, error) {
	return int64(len(m.all)), nil
}

var _ model.AlbumRepository = (*MockAlbumRepo)(nil)

package persistence

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/deluan/gosonic/domain"
)

func CreateMockAlbumRepo() *MockAlbum {
	return &MockAlbum{}
}

type MockAlbum struct {
	domain.AlbumRepository
	data    map[string]*domain.Album
	all     domain.Albums
	err     bool
	Options domain.QueryOptions
}

func (m *MockAlbum) SetError(err bool) {
	m.err = err
}

func (m *MockAlbum) SetData(j string, size int) {
	m.data = make(map[string]*domain.Album)
	m.all = make(domain.Albums, size)
	err := json.Unmarshal([]byte(j), &m.all)
	if err != nil {
		fmt.Println("ERROR: ", err)
	}
	for _, a := range m.all {
		m.data[a.Id] = &a
	}
}

func (m *MockAlbum) Exists(id string) (bool, error) {
	_, found := m.data[id]
	return found, nil
}

func (m *MockAlbum) Get(id string) (*domain.Album, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	if d, ok := m.data[id]; ok {
		return d, nil
	}
	return nil, domain.ErrNotFound
}

func (m *MockAlbum) GetAll(qo domain.QueryOptions) (*domain.Albums, error) {
	m.Options = qo
	if m.err {
		return nil, errors.New("Error!")
	}
	return &m.all, nil
}

func (m *MockAlbum) FindByArtist(artistId string) (*domain.Albums, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	var res = make(domain.Albums, len(m.data))
	i := 0
	for _, a := range m.data {
		if a.ArtistId == artistId {
			res[i] = *a
			i++
		}
	}

	return &res, nil
}

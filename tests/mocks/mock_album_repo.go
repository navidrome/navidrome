package mocks

import (
	"encoding/json"
	"fmt"
	"github.com/deluan/gosonic/domain"
"errors"
)

func CreateMockAlbumRepo() *MockAlbum {
	return &MockAlbum{}
}

type MockAlbum struct {
	domain.AlbumRepository
	data map[string]*domain.Album
	err  bool
}

func (m *MockAlbum) SetError(err bool) {
	m.err = err
}

func (m *MockAlbum) SetData(j string, size int) {
	m.data = make(map[string]*domain.Album)
	var l = make([]domain.Album, size)
	err := json.Unmarshal([]byte(j), &l)
	if err != nil {
		fmt.Println("ERROR: ", err)
	}
	for _, a := range l {
		m.data[a.Id] = &a
	}
}

func (m *MockAlbum) Exists(id string) (bool, error) {
	if m.err {
		return false, errors.New("Error!")
	}
	_, found := m.data[id];
	return found, nil
}

func (m *MockAlbum) Get(id string) (*domain.Album, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	return m.data[id], nil
}

func (m *MockAlbum) FindByArtist(artistId string) ([]domain.Album, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	var res = make([]domain.Album, len(m.data))
	i := 0
	for _, a := range m.data {
		if a.ArtistId == artistId {
			res[i] = *a
			i++
		}
	}

	return res, nil
}
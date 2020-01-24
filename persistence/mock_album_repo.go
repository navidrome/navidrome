package persistence

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/deluan/navidrome/model"
)

func CreateMockAlbumRepo() *MockAlbum {
	return &MockAlbum{}
}

type MockAlbum struct {
	model.AlbumRepository
	data    map[string]*model.Album
	all     model.Albums
	err     bool
	Options model.QueryOptions
}

func (m *MockAlbum) SetError(err bool) {
	m.err = err
}

func (m *MockAlbum) SetData(j string, size int) {
	m.data = make(map[string]*model.Album)
	m.all = make(model.Albums, size)
	err := json.Unmarshal([]byte(j), &m.all)
	if err != nil {
		fmt.Println("ERROR: ", err)
	}
	for _, a := range m.all {
		m.data[a.ID] = &a
	}
}

func (m *MockAlbum) Exists(id string) (bool, error) {
	_, found := m.data[id]
	return found, nil
}

func (m *MockAlbum) Get(id string) (*model.Album, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	if d, ok := m.data[id]; ok {
		return d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockAlbum) GetAll(qo ...model.QueryOptions) (model.Albums, error) {
	if len(qo) > 0 {
		m.Options = qo[0]
	}
	if m.err {
		return nil, errors.New("Error!")
	}
	return m.all, nil
}

func (m *MockAlbum) FindByArtist(artistId string) (model.Albums, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	var res = make(model.Albums, len(m.data))
	i := 0
	for _, a := range m.data {
		if a.ArtistID == artistId {
			res[i] = *a
			i++
		}
	}

	return res, nil
}

var _ model.AlbumRepository = (*MockAlbum)(nil)

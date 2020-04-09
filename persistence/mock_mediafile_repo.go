package persistence

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/deluan/navidrome/model"
)

func CreateMockMediaFileRepo() *MockMediaFile {
	return &MockMediaFile{}
}

type MockMediaFile struct {
	model.MediaFileRepository
	data map[string]model.MediaFile
	err  bool
}

func (m *MockMediaFile) SetError(err bool) {
	m.err = err
}

func (m *MockMediaFile) SetData(j string) {
	m.data = make(map[string]model.MediaFile)
	var l = model.MediaFiles{}
	err := json.Unmarshal([]byte(j), &l)
	if err != nil {
		fmt.Println("ERROR: ", err)
	}
	for _, a := range l {
		m.data[a.ID] = a
	}
}

func (m *MockMediaFile) Exists(id string) (bool, error) {
	if m.err {
		return false, errors.New("Error!")
	}
	_, found := m.data[id]
	return found, nil
}

func (m *MockMediaFile) Get(id string) (*model.MediaFile, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	if d, ok := m.data[id]; ok {
		return &d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockMediaFile) FindByAlbum(artistId string) (model.MediaFiles, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	var res = make(model.MediaFiles, len(m.data))
	i := 0
	for _, a := range m.data {
		if a.AlbumID == artistId {
			res[i] = a
			i++
		}
	}

	return res, nil
}

var _ model.MediaFileRepository = (*MockMediaFile)(nil)

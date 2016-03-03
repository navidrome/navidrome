package mocks

import (
	"encoding/json"
	"fmt"
	"github.com/deluan/gosonic/domain"
"errors"
)

func CreateMockMediaFileRepo() *MockMediaFile {
	return &MockMediaFile{}
}

type MockMediaFile struct {
	domain.MediaFileRepository
	data map[string]*domain.MediaFile
	err  bool
}

func (m *MockMediaFile) SetError(err bool) {
	m.err = err
}

func (m *MockMediaFile) SetData(j string, size int) {
	m.data = make(map[string]*domain.MediaFile)
	var l = make([]domain.MediaFile, size)
	err := json.Unmarshal([]byte(j), &l)
	if err != nil {
		fmt.Println("ERROR: ", err)
	}
	for _, a := range l {
		m.data[a.Id] = &a
	}
}

func (m *MockMediaFile) Exists(id string) (bool, error) {
	if m.err {
		return false, errors.New("Error!")
	}
	_, found := m.data[id];
	return found, nil
}

func (m *MockMediaFile) Get(id string) (*domain.MediaFile, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	return m.data[id], nil
}

func (m *MockMediaFile) FindByAlbum(artistId string) ([]domain.MediaFile, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	var res = make([]domain.MediaFile, len(m.data))
	i := 0
	for _, a := range m.data {
		if a.AlbumId == artistId {
			res[i] = *a
			i++
		}
	}

	return res, nil
}
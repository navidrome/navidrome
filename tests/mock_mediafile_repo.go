package tests

import (
	"errors"

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

func (m *MockMediaFile) SetData(mfs model.MediaFiles) {
	m.data = make(map[string]model.MediaFile)
	for _, mf := range mfs {
		m.data[mf.ID] = mf
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

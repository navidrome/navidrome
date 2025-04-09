package tests

import (
	"github.com/navidrome/navidrome/model"
	"golang.org/x/exp/maps"
)

type MockLibraryRepo struct {
	model.LibraryRepository
	Data map[int]model.Library
	Err  error
}

func (m *MockLibraryRepo) SetData(data model.Libraries) {
	m.Data = make(map[int]model.Library)
	for _, d := range data {
		m.Data[d.ID] = d
	}
}

func (m *MockLibraryRepo) GetAll(...model.QueryOptions) (model.Libraries, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return maps.Values(m.Data), nil
}

func (m *MockLibraryRepo) GetPath(id int) (string, error) {
	if m.Err != nil {
		return "", m.Err
	}
	if lib, ok := m.Data[id]; ok {
		return lib.Path, nil
	}
	return "", model.ErrNotFound
}

var _ model.LibraryRepository = &MockLibraryRepo{}

package tests

import (
	"github.com/navidrome/navidrome/model"
	"golang.org/x/exp/maps"
)

type MockLibraryRepo struct {
	model.LibraryRepository
	data map[int]model.Library
	Err  error
}

func (m *MockLibraryRepo) SetData(data model.Libraries) {
	m.data = make(map[int]model.Library)
	for _, d := range data {
		m.data[d.ID] = d
	}
}

func (m *MockLibraryRepo) GetAll(...model.QueryOptions) (model.Libraries, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return maps.Values(m.data), nil
}

func (m *MockLibraryRepo) GetPath(id int) (string, error) {
	if m.Err != nil {
		return "", m.Err
	}
	if lib, ok := m.data[id]; ok {
		return lib.Path, nil
	}
	return "", model.ErrNotFound
}

var _ model.LibraryRepository = &MockLibraryRepo{}

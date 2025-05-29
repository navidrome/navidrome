package tests

import (
	"github.com/navidrome/navidrome/model"
)

type MockLibraryRepo struct {
	model.LibraryRepository
	Data map[int]*model.Library
	Err  error
}

func (m *MockLibraryRepo) SetData(data model.Libraries) {
	m.Data = make(map[int]*model.Library)
	for i, d := range data {
		m.Data[d.ID] = &data[i]
	}
}

func (m *MockLibraryRepo) Get(id int) (*model.Library, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if lib, ok := m.Data[id]; ok {
		return lib, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockLibraryRepo) GetAll(...model.QueryOptions) (model.Libraries, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	var result model.Libraries
	for _, lib := range m.Data {
		result = append(result, *lib)
	}
	return result, nil
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

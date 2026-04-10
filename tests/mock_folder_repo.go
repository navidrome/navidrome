package tests

import (
	"errors"

	"github.com/navidrome/navidrome/model"
)

func CreateMockFolderRepo() *MockFolderRepo {
	return &MockFolderRepo{
		Data: make(map[string]*model.Folder),
	}
}

type MockFolderRepo struct {
	model.FolderRepository
	Data    map[string]*model.Folder
	Err     bool
	Options model.QueryOptions
}

func (m *MockFolderRepo) SetError(err bool) {
	m.Err = err
}

func (m *MockFolderRepo) SetData(folders []model.Folder) {
	m.Data = make(map[string]*model.Folder)
	for i, f := range folders {
		m.Data[f.ID] = &folders[i]
	}
}

func (m *MockFolderRepo) Get(id string) (*model.Folder, error) {
	if m.Err {
		return nil, errors.New("Error!")
	}
	if d, ok := m.Data[id]; ok {
		return d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockFolderRepo) GetAll(opts ...model.QueryOptions) ([]model.Folder, error) {
	if m.Err {
		return nil, errors.New("Error!")
	}
	if len(opts) > 0 {
		m.Options = opts[0]
	}
	var result []model.Folder
	for _, f := range m.Data {
		result = append(result, *f)
	}
	return result, nil
}

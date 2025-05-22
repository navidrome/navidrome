package tests

import (
	"time"

	"github.com/navidrome/navidrome/model"
)

type MockFolderRepo struct {
	model.FolderRepository
	Data    map[string]*model.Folder
	All     []model.Folder
	Options model.QueryOptions
}

func CreateMockFolderRepo() *MockFolderRepo {
	return &MockFolderRepo{
		Data: make(map[string]*model.Folder),
	}
}

func (m *MockFolderRepo) SetData(folders []model.Folder) {
	m.Data = make(map[string]*model.Folder, len(folders))
	m.All = folders
	for i := range m.All {
		m.Data[m.All[i].ID] = &m.All[i]
	}
}

func (m *MockFolderRepo) Get(id string) (*model.Folder, error) {
	if d, ok := m.Data[id]; ok {
		return d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockFolderRepo) GetByPath(lib model.Library, path string) (*model.Folder, error) {
	for _, f := range m.All {
		if f.LibraryID == lib.ID && f.Path+"/"+f.Name == path {
			return &f, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *MockFolderRepo) GetAll(qo ...model.QueryOptions) ([]model.Folder, error) {
	if len(qo) > 0 {
		m.Options = qo[0]
	}
	return m.All, nil
}

func (m *MockFolderRepo) CountAll(...model.QueryOptions) (int64, error) {
	return int64(len(m.All)), nil
}

func (m *MockFolderRepo) Put(folder *model.Folder) error {
	m.Data[folder.ID] = folder
	return nil
}

func (m *MockFolderRepo) GetLastUpdates(lib model.Library) (map[string]time.Time, error) {
	updates := make(map[string]time.Time)
	for _, f := range m.All {
		if f.LibraryID == lib.ID {
			updates[f.Path+"/"+f.Name] = f.UpdateAt
		}
	}
	return updates, nil
}

func (m *MockFolderRepo) MarkMissing(missing bool, ids ...string) error {
	for _, id := range ids {
		if f, ok := m.Data[id]; ok {
			f.Missing = missing
		}
	}
	return nil
}

func (m *MockFolderRepo) GetTouchedWithPlaylists() (model.FolderCursor, error) {
	return func(yield func(model.Folder, error) bool) {
		for _, f := range m.All {
			if !yield(f, nil) {
				break
			}
		}
	}, nil
}

var _ model.FolderRepository = (*MockFolderRepo)(nil)
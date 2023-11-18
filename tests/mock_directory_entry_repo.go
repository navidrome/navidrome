package tests

import "github.com/navidrome/navidrome/model"

func CreateMockDirectoryEntryRepo() *MockDirectoryEntryRepo {
	return &MockDirectoryEntryRepo{
		Data: map[string]*model.DirectoryEntry{},
	}
}

type MockDirectoryEntryRepo struct {
	model.MediaFileRepository
	Error error
	Data  map[string]*model.DirectoryEntry
}

func (m *MockDirectoryEntryRepo) BrowserDirectory(id string) (model.DirectoryEntiesOrFiles, error) {
	return nil, nil
}

func (m *MockDirectoryEntryRepo) Delete(id string) error {
	if m.Error != nil {
		return m.Error
	}
	delete(m.Data, id)
	return nil
}

func (m *MockDirectoryEntryRepo) Get(id string) (*model.DirectoryEntry, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	data, ok := m.Data[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return data, nil
}

func (m *MockDirectoryEntryRepo) GetDbRoot() (model.DirectoryEntries, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	folders := model.DirectoryEntries{}
	for _, folder := range m.Data {
		if folder.ParentId == "" {
			folders = append(folders, *folder)
		}
	}

	return folders, nil
}

func (m *MockDirectoryEntryRepo) GetAllDirectories() (model.DirectoryEntries, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	folders := model.DirectoryEntries{}
	for _, folder := range m.Data {
		folders = append(folders, *folder)
	}

	return folders, nil
}

func (m *MockDirectoryEntryRepo) Put(folder *model.DirectoryEntry) error {
	if m.Error != nil {
		return m.Error
	}
	m.Data[folder.ID] = folder
	return nil
}

var _ model.DirectoryEntryRepository = (*MockDirectoryEntryRepo)(nil)

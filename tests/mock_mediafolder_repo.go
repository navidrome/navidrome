package tests

import "github.com/navidrome/navidrome/model"

func CreateMockMediaFolderRepo() *MockMediaFolderRepo {
	return &MockMediaFolderRepo{
		Data: map[string]*model.MediaFolder{},
	}
}

type MockMediaFolderRepo struct {
	model.MediaFileRepository
	Error error
	Data  map[string]*model.MediaFolder
}

func (m *MockMediaFolderRepo) BrowserDirectory(id string) (model.MediaFolderOrFiles, error) {
	return nil, nil
}

func (m *MockMediaFolderRepo) Delete(id string) error {
	if m.Error != nil {
		return m.Error
	}
	delete(m.Data, id)
	return nil
}

func (m *MockMediaFolderRepo) Get(id string) (*model.MediaFolder, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	data, ok := m.Data[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return data, nil
}

func (m *MockMediaFolderRepo) GetDbRoot() (model.MediaFolders, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	folders := model.MediaFolders{}
	for _, folder := range m.Data {
		if folder.ParentId == "" {
			folders = append(folders, *folder)
		}
	}

	return folders, nil
}

func (m *MockMediaFolderRepo) GetRoot() (model.MediaFolders, error) {
	return m.GetDbRoot()
}

func (m *MockMediaFolderRepo) GetAllDirectories() (model.MediaFolders, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	folders := model.MediaFolders{}
	for _, folder := range m.Data {
		folders = append(folders, *folder)
	}

	return folders, nil
}

func (m *MockMediaFolderRepo) Put(folder *model.MediaFolder) error {
	if m.Error != nil {
		return m.Error
	}
	m.Data[folder.ID] = folder
	return nil
}

var _ model.MediaFolderRepository = (*MockMediaFolderRepo)(nil)

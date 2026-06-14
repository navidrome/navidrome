package tests

import "github.com/navidrome/navidrome/model"

type MockFolderRepo struct {
	model.FolderRepository
	TouchAllErr    error
	TouchAllCalled bool
}

func (m *MockFolderRepo) TouchAllWithPlaylists() error {
	m.TouchAllCalled = true
	return m.TouchAllErr
}

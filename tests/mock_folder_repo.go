package tests

import (
	"sync"

	"github.com/navidrome/navidrome/model"
)

type MockFolderRepo struct {
	model.FolderRepository
	mu            sync.Mutex
	TouchAllErr   error
	TouchAllCount int64
	touchAllCalls int
}

func (m *MockFolderRepo) TouchAllWithPlaylists() (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.touchAllCalls++
	return m.TouchAllCount, m.TouchAllErr
}

// TouchAllCallCount returns how many times TouchAllWithPlaylists was called.
// Safe to call concurrently with the background scan trigger.
func (m *MockFolderRepo) TouchAllCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.touchAllCalls
}

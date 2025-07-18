package core

import (
	"context"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
)

// MockLibraryWrapper provides a simple wrapper around MockLibraryRepo
// that implements the core.Library interface for testing
type MockLibraryWrapper struct {
	*tests.MockLibraryRepo
}

// MockLibraryRestAdapter adapts MockLibraryRepo to rest.Repository interface
type MockLibraryRestAdapter struct {
	*tests.MockLibraryRepo
}

// NewMockLibraryService creates a new mock library service for testing
func NewMockLibraryService() Library {
	repo := &tests.MockLibraryRepo{
		Data: make(map[int]model.Library),
	}
	// Set up default test data
	repo.SetData(model.Libraries{
		{ID: 1, Name: "Test Library 1", Path: "/music/library1"},
		{ID: 2, Name: "Test Library 2", Path: "/music/library2"},
	})
	return &MockLibraryWrapper{MockLibraryRepo: repo}
}

func (m *MockLibraryWrapper) NewRepository(ctx context.Context) rest.Repository {
	return &MockLibraryRestAdapter{MockLibraryRepo: m.MockLibraryRepo}
}

// rest.Repository interface implementation

func (a *MockLibraryRestAdapter) Delete(id string) error {
	return a.DeleteByStringID(id)
}

var _ Library = (*MockLibraryWrapper)(nil)
var _ rest.Repository = (*MockLibraryRestAdapter)(nil)

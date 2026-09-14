package tests

import (
	"context"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
)

// MockLibraryService provides a simple wrapper around MockLibraryRepo
// that implements the core.Library interface for testing.
// Returns concrete type to avoid import cycles - callers assign to core.Library.
type MockLibraryService struct {
	*MockLibraryRepo
}

// MockLibraryRestAdapter adapts MockLibraryRepo to the REST repository interface
type MockLibraryRestAdapter struct {
	*MockLibraryRepo
}

// NewMockLibraryService creates a new mock library service for testing.
// Returns concrete type - assign to core.Library at call site.
func NewMockLibraryService() *MockLibraryService {
	repo := &MockLibraryRepo{
		Data: make(map[int]model.Library),
	}
	// Set up default test data
	repo.SetData(model.Libraries{
		{ID: 1, Name: "Test Library 1", Path: "/music/library1"},
		{ID: 2, Name: "Test Library 2", Path: "/music/library2"},
	})
	return &MockLibraryService{MockLibraryRepo: repo}
}

func (m *MockLibraryService) NewRepository(ctx context.Context) rest.Repository[model.Library] {
	return &MockLibraryRestAdapter{MockLibraryRepo: m.MockLibraryRepo}
}

func (a *MockLibraryRestAdapter) Delete(_ context.Context, ids ...string) error {
	for _, id := range ids {
		if err := a.DeleteByStringID(id); err != nil {
			return err
		}
	}
	return nil
}

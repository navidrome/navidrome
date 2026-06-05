package tests

import (
	"context"

	"github.com/deluan/rest"
)

// MockUserService provides a simple wrapper around MockedUserRepo
// that implements the core.User interface for testing.
// Returns concrete type to avoid import cycles - callers assign to core.User.
type MockUserService struct {
	*MockedUserRepo
}

// MockUserRestAdapter adapts MockedUserRepo to rest.Repository interface
type MockUserRestAdapter struct {
	*MockedUserRepo
}

// NewMockUserService creates a new mock user service for testing.
// Returns concrete type - assign to core.User at call site.
func NewMockUserService() *MockUserService {
	repo := CreateMockUserRepo()
	return &MockUserService{MockedUserRepo: repo}
}

func (m *MockUserService) NewRepository(ctx context.Context) rest.Repository {
	return &MockUserRestAdapter{MockedUserRepo: m.MockedUserRepo}
}

package tests

import (
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
)

// MockUserService provides a simple wrapper around MockedUserRepo
// that implements the core.User interface for testing.
// Returns concrete type to avoid import cycles - callers assign to core.User.
type MockUserService struct {
	*MockedUserRepo
}

// MockUserRestAdapter adapts MockedUserRepo to the REST repository interface
type MockUserRestAdapter struct {
	*MockedUserRepo
}

// NewMockUserService creates a new mock user service for testing.
// Returns concrete type - assign to core.User at call site.
func NewMockUserService() *MockUserService {
	repo := CreateMockUserRepo()
	return &MockUserService{MockedUserRepo: repo}
}

func (m *MockUserService) Repository() rest.Repository[model.User] {
	return &MockUserRestAdapter{MockedUserRepo: m.MockedUserRepo}
}

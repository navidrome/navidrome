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

func (m *MockUserService) GetUserFeaturePermissions(ctx context.Context, userID string) (map[string]bool, error) {
	return m.MockedUserRepo.GetUserFeaturePermissions(userID)
}

func (m *MockUserService) SetUserFeaturePermissions(ctx context.Context, userID string, permissions map[string]bool) (map[string]bool, error) {
	if err := m.MockedUserRepo.SetUserFeaturePermissions(userID, permissions); err != nil {
		return nil, err
	}
	return m.MockedUserRepo.GetUserFeaturePermissions(userID)
}

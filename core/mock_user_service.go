package core

import (
	"context"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
)

// MockUserWrapper provides a simple wrapper around MockedUserRepo
// that implements the core.User interface for testing
type MockUserWrapper struct {
	*tests.MockedUserRepo
}

// MockUserRestAdapter adapts MockedUserRepo to rest.Repository interface
type MockUserRestAdapter struct {
	*tests.MockedUserRepo
}

// NewMockUserService creates a new mock user service for testing
func NewMockUserService() User {
	repo := tests.CreateMockUserRepo()
	return &MockUserWrapper{MockedUserRepo: repo}
}

func (m *MockUserWrapper) NewRepository(ctx context.Context) rest.Repository {
	return &MockUserRestAdapter{MockedUserRepo: m.MockedUserRepo}
}

// rest.Repository interface implementation

func (a *MockUserRestAdapter) Count(options ...rest.QueryOptions) (int64, error) {
	return a.CountAll()
}

func (a *MockUserRestAdapter) Read(id string) (interface{}, error) {
	return a.Get(id)
}

func (a *MockUserRestAdapter) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return a.GetAll()
}

func (a *MockUserRestAdapter) EntityName() string {
	return "user"
}

func (a *MockUserRestAdapter) NewInstance() interface{} {
	return &model.User{}
}

var _ User = (*MockUserWrapper)(nil)
var _ rest.Repository = (*MockUserRestAdapter)(nil)

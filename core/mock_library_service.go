package core

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
)

// MockLibraryService provides a mock implementation of core.Library interface
// that can be used in tests to prevent nil pointer panics
type MockLibraryService struct {
	Library
	Err       error
	Libraries model.Libraries
}

// NewMockLibraryService creates a new mock library service with some default test data
func NewMockLibraryService() *MockLibraryService {
	return &MockLibraryService{
		Libraries: model.Libraries{
			{ID: 1, Name: "Test Library 1", Path: "/music/library1"},
			{ID: 2, Name: "Test Library 2", Path: "/music/library2"},
		},
	}
}

func (m *MockLibraryService) GetAll(ctx context.Context) (model.Libraries, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Libraries, nil
}

func (m *MockLibraryService) Get(ctx context.Context, id int) (*model.Library, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	for _, lib := range m.Libraries {
		if lib.ID == id {
			return &lib, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *MockLibraryService) Create(ctx context.Context, library *model.Library) error {
	if m.Err != nil {
		return m.Err
	}
	if library.Name == "" {
		return fmt.Errorf("%w: library name is required", model.ErrValidation)
	}
	if library.Path == "" {
		return fmt.Errorf("%w: library path is required", model.ErrValidation)
	}
	// Add to mock data
	library.ID = len(m.Libraries) + 1
	m.Libraries = append(m.Libraries, *library)
	return nil
}

func (m *MockLibraryService) Update(ctx context.Context, library *model.Library) error {
	if m.Err != nil {
		return m.Err
	}
	if library.Name == "" {
		return fmt.Errorf("%w: library name is required", model.ErrValidation)
	}
	if library.Path == "" {
		return fmt.Errorf("%w: library path is required", model.ErrValidation)
	}

	// Find and update in mock data
	for i, lib := range m.Libraries {
		if lib.ID == library.ID {
			m.Libraries[i] = *library
			return nil
		}
	}
	return model.ErrNotFound
}

func (m *MockLibraryService) Delete(ctx context.Context, id int) error {
	if m.Err != nil {
		return m.Err
	}

	// Find and remove from mock data
	for i, lib := range m.Libraries {
		if lib.ID == id {
			m.Libraries = append(m.Libraries[:i], m.Libraries[i+1:]...)
			return nil
		}
	}
	return model.ErrNotFound
}

func (m *MockLibraryService) GetUserLibraries(ctx context.Context, userID string) (model.Libraries, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if userID == "non-existent" {
		return nil, model.ErrNotFound
	}
	// Return all libraries for simplicity in tests
	return m.Libraries, nil
}

func (m *MockLibraryService) SetUserLibraries(ctx context.Context, userID string, libraryIDs []int) error {
	if m.Err != nil {
		return m.Err
	}
	if userID == "non-existent" {
		return model.ErrNotFound
	}
	if userID == "admin-1" {
		return fmt.Errorf("%w: cannot manually assign libraries to admin users", model.ErrValidation)
	}
	if len(libraryIDs) == 0 {
		return fmt.Errorf("%w: at least one library must be assigned to non-admin users", model.ErrValidation)
	}
	// Validate all library IDs exist
	for _, id := range libraryIDs {
		found := false
		for _, lib := range m.Libraries {
			if lib.ID == id {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%w: library ID %d does not exist", model.ErrValidation, id)
		}
	}
	return nil
}

func (m *MockLibraryService) ValidateLibraryAccess(ctx context.Context, userID string, libraryID int) error {
	if m.Err != nil {
		return m.Err
	}
	// For testing purposes, allow access to all libraries
	return nil
}

func (m *MockLibraryService) NewRepository(ctx context.Context) rest.Repository {
	return &mockLibraryRepository{service: m, ctx: ctx}
}

// mockLibraryRepository provides a REST repository wrapper for the mock service
type mockLibraryRepository struct {
	service *MockLibraryService
	ctx     context.Context
}

func (r *mockLibraryRepository) Count(options ...rest.QueryOptions) (int64, error) {
	libs, err := r.service.GetAll(r.ctx)
	if err != nil {
		return 0, err
	}
	return int64(len(libs)), nil
}

func (r *mockLibraryRepository) Read(id string) (interface{}, error) {
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return nil, rest.ErrNotFound
	}
	lib, err := r.service.Get(r.ctx, idInt)
	if errors.Is(err, model.ErrNotFound) {
		return nil, rest.ErrNotFound
	}
	return lib, err
}

func (r *mockLibraryRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.service.GetAll(r.ctx)
}

func (r *mockLibraryRepository) EntityName() string {
	return "library"
}

func (r *mockLibraryRepository) NewInstance() interface{} {
	return &model.Library{}
}

func (r *mockLibraryRepository) Save(entity interface{}) (string, error) {
	lib := entity.(*model.Library)
	err := r.service.Create(r.ctx, lib)
	if errors.Is(err, model.ErrValidation) {
		return "", &rest.ValidationError{Errors: map[string]string{"validation": err.Error()}}
	}
	if err != nil {
		return "", err
	}
	return strconv.Itoa(lib.ID), nil
}

func (r *mockLibraryRepository) Update(id string, entity interface{}, cols ...string) error {
	lib := entity.(*model.Library)
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return &rest.ValidationError{Errors: map[string]string{"id": "invalid library ID format"}}
	}
	lib.ID = idInt
	err = r.service.Update(r.ctx, lib)
	if errors.Is(err, model.ErrNotFound) {
		return rest.ErrNotFound
	}
	if errors.Is(err, model.ErrValidation) {
		return &rest.ValidationError{Errors: map[string]string{"validation": err.Error()}}
	}
	return err
}

func (r *mockLibraryRepository) Delete(id string) error {
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return &rest.ValidationError{Errors: map[string]string{"id": "invalid library ID format"}}
	}
	err = r.service.Delete(r.ctx, idInt)
	if errors.Is(err, model.ErrNotFound) {
		return rest.ErrNotFound
	}
	return err
}

var _ Library = (*MockLibraryService)(nil)
var _ rest.Repository = (*mockLibraryRepository)(nil)

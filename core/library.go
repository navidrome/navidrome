package core

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

// Library provides business logic for library management and user-library associations
type Library interface {
	GetUserLibraries(ctx context.Context, userID string) (model.Libraries, error)
	SetUserLibraries(ctx context.Context, userID string, libraryIDs []int) error
	ValidateLibraryAccess(ctx context.Context, userID string, libraryID int) error

	NewRepository(ctx context.Context) rest.Repository
}

type libraryService struct {
	ds model.DataStore
}

// NewLibrary creates a new Library service
func NewLibrary(ds model.DataStore) Library {
	return &libraryService{ds: ds}
}

// User-library association operations

func (s *libraryService) GetUserLibraries(ctx context.Context, userID string) (model.Libraries, error) {
	// Verify user exists
	if _, err := s.ds.User(ctx).Get(userID); err != nil {
		return nil, err
	}

	return s.ds.User(ctx).GetUserLibraries(userID)
}

func (s *libraryService) SetUserLibraries(ctx context.Context, userID string, libraryIDs []int) error {
	// Verify user exists
	user, err := s.ds.User(ctx).Get(userID)
	if err != nil {
		return err
	}

	// Admin users get all libraries automatically - don't allow manual assignment
	if user.IsAdmin {
		return fmt.Errorf("%w: cannot manually assign libraries to admin users", model.ErrValidation)
	}

	// Regular users must have at least one library
	if len(libraryIDs) == 0 {
		return fmt.Errorf("%w: at least one library must be assigned to non-admin users", model.ErrValidation)
	}

	// Validate all library IDs exist
	if len(libraryIDs) > 0 {
		if err := s.validateLibraryIDs(ctx, libraryIDs); err != nil {
			return err
		}
	}

	return s.ds.User(ctx).SetUserLibraries(userID, libraryIDs)
}

func (s *libraryService) ValidateLibraryAccess(ctx context.Context, userID string, libraryID int) error {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	// Admin users have access to all libraries
	if user.IsAdmin {
		return nil
	}

	// Check if user has explicit access to this library
	libraries, err := s.ds.User(ctx).GetUserLibraries(userID)
	if err != nil {
		log.Error(ctx, "Error checking library access", "userID", userID, "libraryID", libraryID, err)
		return fmt.Errorf("error checking library access: %w", err)
	}

	for _, lib := range libraries {
		if lib.ID == libraryID {
			return nil
		}
	}

	return fmt.Errorf("%w: user does not have access to library %d", model.ErrNotAuthorized, libraryID)
}

// REST repository wrapper

func (s *libraryService) NewRepository(ctx context.Context) rest.Repository {
	repo := s.ds.Library(ctx)
	wrapper := &libraryRepositoryWrapper{
		ctx:               ctx,
		LibraryRepository: repo,
		Repository:        repo.(rest.Repository),
		ds:                s.ds,
		service:           s,
	}
	return wrapper
}

type libraryRepositoryWrapper struct {
	rest.Repository
	model.LibraryRepository
	ctx     context.Context
	ds      model.DataStore
	service *libraryService
}

func (r *libraryRepositoryWrapper) Save(entity interface{}) (string, error) {
	lib := entity.(*model.Library)
	if err := r.validateLibrary(lib); err != nil {
		return "", err
	}

	err := r.LibraryRepository.Put(lib)
	if err != nil {
		return "", r.mapError(err)
	}

	return strconv.Itoa(lib.ID), nil
}

func (r *libraryRepositoryWrapper) Update(id string, entity interface{}, cols ...string) error {
	lib := entity.(*model.Library)
	libID, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("invalid library ID: %s", id)
	}

	lib.ID = libID
	if err := r.validateLibrary(lib); err != nil {
		return err
	}

	// Verify library exists
	if _, err := r.Get(libID); err != nil {
		return r.mapError(err)
	}

	err = r.LibraryRepository.Put(lib)
	return r.mapError(err)
}

func (r *libraryRepositoryWrapper) Delete(id string) error {
	libID, err := strconv.Atoi(id)
	if err != nil {
		return rest.ValidationError{Errors: map[string]string{
			"id": "invalid library ID format",
		}}
	}

	err = r.LibraryRepository.Delete(libID)
	return r.mapError(err)
}

// Helper methods

func (r *libraryRepositoryWrapper) mapError(err error) error {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return rest.ErrNotFound
	case errors.Is(err, model.ErrNotAuthorized):
		return rest.ErrPermissionDenied
	default:
		return err
	}
}

func (r *libraryRepositoryWrapper) validateLibrary(library *model.Library) error {
	if library.Name == "" {
		return rest.ValidationError{Errors: map[string]string{
			"name": "library name is required",
		}}
	}

	if library.Path == "" {
		return rest.ValidationError{Errors: map[string]string{
			"path": "library path is required",
		}}
	}

	return nil
}

func (s *libraryService) validateLibraryIDs(ctx context.Context, libraryIDs []int) error {
	if len(libraryIDs) == 0 {
		return nil
	}

	// Use CountAll to efficiently validate library IDs exist
	count, err := s.ds.Library(ctx).CountAll(model.QueryOptions{
		Filters: squirrel.Eq{"id": libraryIDs},
	})
	if err != nil {
		return fmt.Errorf("error validating library IDs: %w", err)
	}

	if int(count) != len(libraryIDs) {
		return fmt.Errorf("%w: one or more library IDs are invalid", model.ErrValidation)
	}

	return nil
}

package core

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

// Library provides business logic for library management and user-library associations
type Library interface {
	// Library CRUD operations
	GetAll(ctx context.Context) (model.Libraries, error)
	Get(ctx context.Context, id int) (*model.Library, error)
	Create(ctx context.Context, library *model.Library) error
	Update(ctx context.Context, library *model.Library) error
	Delete(ctx context.Context, id int) error

	// User-library association operations
	GetUserLibraries(ctx context.Context, userID string) (model.Libraries, error)
	SetUserLibraries(ctx context.Context, userID string, libraryIDs []int) error
	ValidateLibraryAccess(ctx context.Context, userID string, libraryID int) error
}

type libraryService struct {
	ds model.DataStore
}

// NewLibrary creates a new Library service
func NewLibrary(ds model.DataStore) Library {
	return &libraryService{ds: ds}
}

// Library CRUD operations

func (s *libraryService) GetAll(ctx context.Context) (model.Libraries, error) {
	return s.ds.Library(ctx).GetAll()
}

func (s *libraryService) Get(ctx context.Context, id int) (*model.Library, error) {
	return s.ds.Library(ctx).Get(id)
}

func (s *libraryService) Create(ctx context.Context, library *model.Library) error {
	if err := s.validateLibrary(library); err != nil {
		return err
	}

	return s.ds.Library(ctx).Put(library)
}

func (s *libraryService) Update(ctx context.Context, library *model.Library) error {
	if err := s.validateLibrary(library); err != nil {
		return err
	}

	// Verify library exists
	if _, err := s.ds.Library(ctx).Get(library.ID); err != nil {
		return err
	}

	return s.ds.Library(ctx).Put(library)
}

func (s *libraryService) Delete(ctx context.Context, id int) error {
	// Verify library exists
	if _, err := s.ds.Library(ctx).Get(id); err != nil {
		return err
	}

	return s.ds.Library(ctx).Delete(id)
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

// Helper methods

func (s *libraryService) validateLibrary(library *model.Library) error {
	if library.Name == "" {
		return fmt.Errorf("%w: library name is required", model.ErrValidation)
	}
	if library.Path == "" {
		return fmt.Errorf("%w: library path is required", model.ErrValidation)
	}
	return nil
}

func (s *libraryService) validateLibraryIDs(ctx context.Context, libraryIDs []int) error {
	// Use CountAll with IN filter to efficiently check if all library IDs exist
	count, err := s.ds.Library(ctx).CountAll(model.QueryOptions{
		Filters: squirrel.Eq{"id": libraryIDs},
	})
	if err != nil {
		return fmt.Errorf("error validating library IDs: %w", err)
	}

	if int(count) != len(libraryIDs) {
		// Find which library IDs don't exist for better error message
		existingLibraries, err := s.ds.Library(ctx).GetAll(model.QueryOptions{
			Filters: squirrel.Eq{"id": libraryIDs},
		})
		if err != nil {
			return fmt.Errorf("error getting libraries: %w", err)
		}

		libraryMap := make(map[int]bool)
		for _, lib := range existingLibraries {
			libraryMap[lib.ID] = true
		}

		for _, libID := range libraryIDs {
			if !libraryMap[libID] {
				return fmt.Errorf("%w: library ID %d does not exist", model.ErrNotFound, libID)
			}
		}
	}

	return nil
}

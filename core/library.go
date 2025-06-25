package core

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

// Scanner defines the interface for triggering library scans
type Scanner interface {
	ScanAll(ctx context.Context, fullScan bool) (warnings []string, err error)
}

// Library provides business logic for library management and user-library associations
type Library interface {
	GetUserLibraries(ctx context.Context, userID string) (model.Libraries, error)
	SetUserLibraries(ctx context.Context, userID string, libraryIDs []int) error
	ValidateLibraryAccess(ctx context.Context, userID string, libraryID int) error

	NewRepository(ctx context.Context) rest.Repository
}

type libraryService struct {
	ds      model.DataStore
	scanner Scanner
}

// NewLibrary creates a new Library service
func NewLibrary(ds model.DataStore, scanner Scanner) Library {
	return &libraryService{ds: ds, scanner: scanner}
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
		scanner:           s.scanner,
	}
	return wrapper
}

type libraryRepositoryWrapper struct {
	rest.Repository
	model.LibraryRepository
	ctx     context.Context
	ds      model.DataStore
	scanner Scanner
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

	// Trigger scan after successful library creation
	if r.scanner != nil {
		go func() {
			log.Info(r.ctx, "Triggering scan for new library", "libraryID", lib.ID, "name", lib.Name, "path", lib.Path)
			start := time.Now()
			warnings, err := r.scanner.ScanAll(r.ctx, false) // Quick scan for new library
			if err != nil {
				log.Error(r.ctx, "Error scanning new library", "libraryID", lib.ID, "name", lib.Name, err)
			} else {
				log.Info(r.ctx, "Scan completed for new library", "libraryID", lib.ID, "name", lib.Name, "warnings", len(warnings), "elapsed", time.Since(start))
			}
		}()
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

	// Get the original library to check if path changed
	originalLib, err := r.Get(libID)
	if err != nil {
		return r.mapError(err)
	}

	pathChanged := originalLib.Path != lib.Path

	err = r.LibraryRepository.Put(lib)
	if err != nil {
		return r.mapError(err)
	}

	// Trigger scan if path was updated
	if pathChanged && r.scanner != nil {
		go func() {
			log.Info(r.ctx, "Triggering scan for updated library", "libraryID", lib.ID, "name", lib.Name, "oldPath", originalLib.Path, "newPath", lib.Path)
			start := time.Now()
			warnings, err := r.scanner.ScanAll(r.ctx, false) // Quick scan for path change
			if err != nil {
				log.Error(r.ctx, "Error scanning updated library", "libraryID", lib.ID, "name", lib.Name, err)
			} else {
				log.Info(r.ctx, "Scan completed for updated library", "libraryID", lib.ID, "name", lib.Name, "warnings", len(warnings), "elapsed", time.Since(start))
			}
		}()
	}

	return nil
}

func (r *libraryRepositoryWrapper) Delete(id string) error {
	libID, err := strconv.Atoi(id)
	if err != nil {
		return &rest.ValidationError{Errors: map[string]string{
			"id": "invalid library ID format",
		}}
	}

	// Get library info before deletion for logging
	lib, err := r.Get(libID)
	if err != nil {
		return r.mapError(err)
	}

	err = r.LibraryRepository.Delete(libID)
	if err != nil {
		return r.mapError(err)
	}

	// Trigger scan after successful library deletion to clean up orphaned data
	if r.scanner != nil {
		go func() {
			log.Info(r.ctx, "Triggering scan after library deletion", "libraryID", lib.ID, "name", lib.Name, "path", lib.Path)
			start := time.Now()
			warnings, err := r.scanner.ScanAll(r.ctx, false) // Quick scan to clean up orphaned data
			if err != nil {
				log.Error(r.ctx, "Error scanning after library deletion", "libraryID", lib.ID, "name", lib.Name, err)
			} else {
				log.Info(r.ctx, "Scan completed after library deletion", "libraryID", lib.ID, "name", lib.Name, "warnings", len(warnings), "elapsed", time.Since(start))
			}
		}()
	}

	return nil
}

// Helper methods

func (r *libraryRepositoryWrapper) mapError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Handle database constraint violations.
	// TODO: Being tied to react-admin translations is not ideal, but this will probably go away with the new UI/API
	if strings.Contains(errStr, "UNIQUE constraint failed") {
		if strings.Contains(errStr, "library.name") {
			return &rest.ValidationError{Errors: map[string]string{"name": "ra.validation.unique"}}
		}
		if strings.Contains(errStr, "library.path") {
			return &rest.ValidationError{Errors: map[string]string{"path": "ra.validation.unique"}}
		}
	}

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
	validationErrors := make(map[string]string)

	if library.Name == "" {
		validationErrors["name"] = "ra.validation.required"
	}

	if library.Path == "" {
		validationErrors["path"] = "ra.validation.required"
	} else {
		// Validate path format and accessibility
		if err := r.validateLibraryPath(library); err != nil {
			validationErrors["path"] = err.Error()
		}
	}

	if len(validationErrors) > 0 {
		return &rest.ValidationError{Errors: validationErrors}
	}

	return nil
}

func (r *libraryRepositoryWrapper) validateLibraryPath(library *model.Library) error {
	// Validate path format
	if !filepath.IsAbs(library.Path) {
		return fmt.Errorf("library path must be absolute")
	}

	// Clean the path to normalize it
	cleanPath := filepath.Clean(library.Path)
	library.Path = cleanPath

	// Check if path exists and is accessible using storage abstraction
	fileStore, err := storage.For(library.Path)
	if err != nil {
		return fmt.Errorf("invalid storage scheme: %w", err)
	}

	fsys, err := fileStore.FS()
	if err != nil {
		log.Warn(r.ctx, "Error validating library.path", "path", library.Path, err)
		return fmt.Errorf("resources.library.validation.pathInvalid")
	}

	// Check if root directory exists
	info, err := fs.Stat(fsys, ".")
	if err != nil {
		// Parse the error message to check for "not a directory"
		log.Warn(r.ctx, "Error stating library.path", "path", library.Path, err)
		errStr := err.Error()
		if strings.Contains(errStr, "not a directory") ||
			strings.Contains(errStr, "The directory name is invalid.") {
			return fmt.Errorf("resources.library.validation.pathNotDirectory")
		} else if os.IsNotExist(err) {
			return fmt.Errorf("resources.library.validation.pathNotFound")
		} else if os.IsPermission(err) {
			return fmt.Errorf("resources.library.validation.pathNotAccessible")
		} else {
			return fmt.Errorf("resources.library.validation.pathInvalid")
		}
	}

	if !info.IsDir() {
		return fmt.Errorf("resources.library.validation.pathNotDirectory")
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

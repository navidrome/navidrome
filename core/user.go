package core

import (
	"context"
	"fmt"
	"slices"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
)

// PluginUnloader defines the interface for unloading disabled plugins.
// This is satisfied by plugins.Manager but defined here to avoid import cycles.
type PluginUnloader interface {
	UnloadDisabledPlugins(ctx context.Context)
}

// User provides business logic for user management with plugin coordination.
type User interface {
	NewRepository(ctx context.Context) rest.Repository

	// Feature permission operations
	GetUserFeaturePermissions(ctx context.Context, userID string) (map[string]bool, error)
	SetUserFeaturePermissions(ctx context.Context, userID string, permissions map[string]bool) (map[string]bool, error)
}

type userService struct {
	ds            model.DataStore
	pluginManager PluginUnloader
}

// NewUser creates a new User service
func NewUser(ds model.DataStore, pluginManager PluginUnloader) User {
	return &userService{
		ds:            ds,
		pluginManager: pluginManager,
	}
}

// NewRepository returns a REST repository wrapper for user operations.
// The wrapper intercepts Delete operations to coordinate plugin unloading.
func (s *userService) NewRepository(ctx context.Context) rest.Repository {
	repo := s.ds.User(ctx)
	wrapper := &userRepositoryWrapper{
		ctx:            ctx,
		UserRepository: repo,
		pluginManager:  s.pluginManager,
	}
	return wrapper
}

type userRepositoryWrapper struct {
	model.UserRepository
	ctx           context.Context
	pluginManager PluginUnloader
}

// Save implements rest.Persistable by delegating to the underlying repository.
func (r *userRepositoryWrapper) Save(entity any) (string, error) {
	return r.UserRepository.(rest.Persistable).Save(entity)
}

// Update implements rest.Persistable by delegating to the underlying repository.
func (r *userRepositoryWrapper) Update(id string, entity any, cols ...string) error {
	return r.UserRepository.(rest.Persistable).Update(id, entity, cols...)
}

// Feature permission operations

func (s *userService) GetUserFeaturePermissions(ctx context.Context, userID string) (map[string]bool, error) {
	// Verify user exists
	if _, err := s.ds.User(ctx).Get(userID); err != nil {
		return nil, err
	}
	return s.ds.User(ctx).GetUserFeaturePermissions(userID)
}

func (s *userService) SetUserFeaturePermissions(ctx context.Context, userID string, permissions map[string]bool) (map[string]bool, error) {
	user, err := s.ds.User(ctx).Get(userID)
	if err != nil {
		return nil, err
	}

	// Admins bypass every feature gate (see persistence.hasFeaturePermission) - don't allow
	// manual restriction that would have no effect and would only confuse the admin UI.
	if user.IsAdmin {
		return nil, fmt.Errorf("%w: cannot restrict feature access for admin users", model.ErrValidation)
	}

	for feature := range permissions {
		if !slices.Contains(model.AllUserFeatures, feature) {
			return nil, fmt.Errorf("%w: unknown feature %q", model.ErrValidation, feature)
		}
	}

	if err := s.ds.User(ctx).SetUserFeaturePermissions(userID, permissions); err != nil {
		return nil, err
	}
	return s.ds.User(ctx).GetUserFeaturePermissions(userID)
}

// Delete implements rest.Persistable and coordinates plugin unloading.
func (r *userRepositoryWrapper) Delete(id string) error {
	// The underlying repository Delete handles the database cleanup
	// including calling cleanupPluginUserReferences
	err := r.UserRepository.(rest.Persistable).Delete(id)
	if err != nil {
		return err
	}

	// After successful deletion, check if any plugins were auto-disabled
	// and need to be unloaded from memory
	r.pluginManager.UnloadDisabledPlugins(r.ctx)

	return nil
}

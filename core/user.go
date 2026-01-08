package core

import (
	"context"

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
func (r *userRepositoryWrapper) Save(entity interface{}) (string, error) {
	return r.UserRepository.(rest.Persistable).Save(entity)
}

// Update implements rest.Persistable by delegating to the underlying repository.
func (r *userRepositoryWrapper) Update(id string, entity interface{}, cols ...string) error {
	return r.UserRepository.(rest.Persistable).Update(id, entity, cols...)
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

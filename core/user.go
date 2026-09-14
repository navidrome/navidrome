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
	NewRepository(ctx context.Context) rest.Repository[model.User]
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
func (s *userService) NewRepository(ctx context.Context) rest.Repository[model.User] {
	return &userRepositoryWrapper{
		UserRepository: s.ds.User(),
		pluginManager:  s.pluginManager,
	}
}

type userRepositoryWrapper struct {
	model.UserRepository
	pluginManager PluginUnloader
}

// Delete coordinates plugin unloading after the repository cleans up the database.
func (r *userRepositoryWrapper) Delete(ctx context.Context, ids ...string) error {
	if err := r.UserRepository.Delete(ctx, ids...); err != nil {
		return err
	}
	r.pluginManager.UnloadDisabledPlugins(ctx)
	return nil
}

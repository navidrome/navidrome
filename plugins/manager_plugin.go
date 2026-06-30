package plugins

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	extism "github.com/extism/go-sdk"
	"github.com/navidrome/navidrome/model"
	"github.com/tetratelabs/wazero"
)

// plugin represents a loaded plugin
type plugin struct {
	name           string // Plugin name (from filename)
	path           string // Path to the wasm file
	manifest       *Manifest
	compiled       *extism.CompiledPlugin
	capabilities   []Capability // Auto-detected capabilities based on exported functions
	closers        []io.Closer  // Cleanup functions to call on unload
	metrics        PluginMetricsRecorder
	allowedUserIDs []string // User IDs this plugin can access (from DB configuration)
	allUsers       bool     // If true, plugin can access all users
	libraries      libraryAccess
}

// instance creates a new plugin instance for the given context.
// The context is used for cancellation - if cancelled during a call,
// the module will be terminated and the instance becomes unusable.
func (p *plugin) instance(ctx context.Context) (*extism.Plugin, error) {
	instance, err := p.compiled.Instance(ctx, extism.PluginInstanceConfig{
		ModuleConfig: wazero.NewModuleConfig().WithSysWalltime().WithRandSource(rand.Reader),
	})
	if err != nil {
		return nil, err
	}
	instance.SetLogger(extismLogger(p.name))
	return instance, nil
}

func (p *plugin) Close() error {
	var errs []error
	for _, f := range p.closers {
		err := f.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (p *plugin) hasLibraryFilesystemAccess(libID int) bool {
	return p.manifest.HasLibraryFilesystemPermission() && p.libraries.contains(libID)
}

// libraryAccess captures the set of libraries a plugin is permitted to see,
// precomputed at load time for O(1) lookup.
type libraryAccess struct {
	allLibraries bool
	libraryIDSet map[int]struct{}
}

func newLibraryAccess(allowedLibraryIDs []int, allLibraries bool) libraryAccess {
	set := make(map[int]struct{}, len(allowedLibraryIDs))
	for _, id := range allowedLibraryIDs {
		set[id] = struct{}{}
	}
	return libraryAccess{allLibraries: allLibraries, libraryIDSet: set}
}

func (a libraryAccess) contains(libID int) bool {
	if a.allLibraries {
		return true
	}
	_, ok := a.libraryIDSet[libID]
	return ok
}

// userAccess captures the set of users a plugin is permitted to act as,
// precomputed at load time for O(1) lookup.
type userAccess struct {
	allUsers  bool
	userIDSet map[string]struct{}
}

func newUserAccess(allowedUserIDs []string, allUsers bool) userAccess {
	set := make(map[string]struct{}, len(allowedUserIDs))
	for _, id := range allowedUserIDs {
		set[id] = struct{}{}
	}
	return userAccess{allUsers: allUsers, userIDSet: set}
}

// allows reports whether the plugin may act as the given user ID.
func (a userAccess) allows(userID string) bool {
	if a.allUsers {
		return true
	}
	_, ok := a.userIDSet[userID]
	return ok
}

// resolve looks up a user by username and authorizes it against this access set.
// It distinguishes a genuinely absent user from a backend failure (the latter is
// surfaced wrapped, not masked as "not found"), and rejects a real user the
// plugin is not permitted to act as.
func (a userAccess) resolve(ctx context.Context, ds model.DataStore, username string) (*model.User, error) {
	usr, err := ds.User(ctx).FindByUsername(username)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, fmt.Errorf("user %q not found", username)
		}
		return nil, fmt.Errorf("looking up user %q: %w", username, err)
	}
	if !a.allows(usr.ID) {
		return nil, fmt.Errorf("plugin is not allowed to act as user %q", username)
	}
	return usr, nil
}

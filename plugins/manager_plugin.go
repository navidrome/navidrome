package plugins

import (
	"context"
	"crypto/rand"
	"errors"
	"io"

	extism "github.com/extism/go-sdk"
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

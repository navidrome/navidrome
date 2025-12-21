package core

import (
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/scrobbler"
)

// TODO(PLUGINS): Remove NoopPluginLoader when real plugin system is implemented

// NoopPluginLoader is a stub implementation of plugin loaders that does nothing.
// This is used as a placeholder until the new plugin system is implemented.
type NoopPluginLoader struct{}

// GetNoopPluginLoader returns a singleton noop plugin loader instance.
func GetNoopPluginLoader() *NoopPluginLoader {
	return &NoopPluginLoader{}
}

// PluginNames returns an empty slice (no plugins available)
func (n *NoopPluginLoader) PluginNames(_ string) []string {
	return nil
}

// LoadMediaAgent returns false (no plugin available)
func (n *NoopPluginLoader) LoadMediaAgent(_ string) (agents.Interface, bool) {
	return nil, false
}

// LoadScrobbler returns false (no plugin available)
func (n *NoopPluginLoader) LoadScrobbler(_ string) (scrobbler.Scrobbler, bool) {
	return nil, false
}

// Verify interface implementations at compile time
var (
	_ agents.PluginLoader    = (*NoopPluginLoader)(nil)
	_ scrobbler.PluginLoader = (*NoopPluginLoader)(nil)
)

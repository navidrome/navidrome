package plugins

import (
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/tetratelabs/wazero"
)

// WasmPlugin is the base interface that all WASM plugins implement
type WasmPlugin interface {
	// PluginName returns the name of the plugin
	PluginName() string
}

// WasmArtistAgent represents a WASM plugin that provides artist metadata
type WasmArtistAgent interface {
	WasmPlugin
	agents.Interface
}

// WasmAlbumAgent represents a WASM plugin that provides album metadata
type WasmAlbumAgent interface {
	WasmPlugin
	agents.Interface
}

// WasmScrobbler represents a WASM plugin that provides scrobbling functionality
type WasmScrobbler interface {
	WasmPlugin
	scrobbler.Scrobbler
}

// PluginCreator is a generic function type for creating plugins
type PluginCreator[P WasmPlugin] func(wasmPath, pluginName string, runtime api.WazeroNewRuntime, mc wazero.ModuleConfig) P

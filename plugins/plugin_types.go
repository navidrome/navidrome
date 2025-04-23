package plugins

import (
	"context"
	"sync"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/tetratelabs/wazero"
)

type pluginTypeInfo struct {
	// loaderCtor is a constructor function that creates a new plugin loader.
	loaderCtor func(context.Context, func(context.Context) (wazero.Runtime, error), wazero.ModuleConfig) (any, error)
	// loadFunc is a function that loads a plugin instance using the given loader.
	// It takes a context, the loader, and the path to the wasm file, returning the plugin instance or an error.
	loadFunc func(context.Context, any, string) (any, error)
	// agentCtor is a constructor function that creates a new agent for the plugin.
	agentCtor func(*sync.Pool, string, string) agents.Interface
}

var pluginTypes = map[string]pluginTypeInfo{
	"ArtistMetadataService": {
		loaderCtor: func(ctx context.Context, runtimeCtor func(context.Context) (wazero.Runtime, error), mc wazero.ModuleConfig) (any, error) {
			return api.NewArtistMetadataServicePlugin(ctx, api.WazeroRuntime(runtimeCtor), api.WazeroModuleConfig(mc))
		},
		loadFunc: func(ctx context.Context, loader any, wasmPath string) (any, error) {
			return loader.(*api.ArtistMetadataServicePlugin).Load(ctx, wasmPath)
		},
		agentCtor: func(pool *sync.Pool, wasmPath, pluginName string) agents.Interface {
			return &wasmArtistAgent{
				wasmBasePlugin: &wasmBasePlugin[api.ArtistMetadataService]{
					pool:     pool,
					wasmPath: wasmPath,
					name:     pluginName,
				},
			}
		},
	},
	"AlbumMetadataService": {
		loaderCtor: func(ctx context.Context, runtimeCtor func(context.Context) (wazero.Runtime, error), mc wazero.ModuleConfig) (any, error) {
			return api.NewAlbumMetadataServicePlugin(ctx, api.WazeroRuntime(runtimeCtor), api.WazeroModuleConfig(mc))
		},
		loadFunc: func(ctx context.Context, loader any, wasmPath string) (any, error) {
			return loader.(*api.AlbumMetadataServicePlugin).Load(ctx, wasmPath)
		},
		agentCtor: func(pool *sync.Pool, wasmPath, pluginName string) agents.Interface {
			return &wasmAlbumAgent{
				wasmBasePlugin: &wasmBasePlugin[api.AlbumMetadataService]{
					pool:     pool,
					wasmPath: wasmPath,
					name:     pluginName,
				},
			}
		},
	},
}

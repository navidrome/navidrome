package plugins

import (
	"context"

	"github.com/navidrome/navidrome/plugins/api"
	"github.com/tetratelabs/wazero"
)

type pluginTypeInfo struct {
	loaderCtor    func(context.Context, func(context.Context) (wazero.Runtime, error), wazero.ModuleConfig) (any, error)
	loadFunc      func(context.Context, any, string) (any, error)
	createAdapter func(any, string, string) any
}

var pluginTypes = map[string]pluginTypeInfo{
	"ArtistMetadataService": {
		loaderCtor: func(ctx context.Context, runtimeCtor func(context.Context) (wazero.Runtime, error), mc wazero.ModuleConfig) (any, error) {
			return api.NewArtistMetadataServicePlugin(ctx, api.WazeroRuntime(runtimeCtor), api.WazeroModuleConfig(mc))
		},
		loadFunc: func(ctx context.Context, loader any, wasmPath string) (any, error) {
			return loader.(*api.ArtistMetadataServicePlugin).Load(ctx, wasmPath)
		},
		createAdapter: func(loader any, wasmPath, pluginName string) any {
			return &wasmArtistAgent{
				wasmBasePlugin: &wasmBasePlugin[api.ArtistMetadataService]{
					wasmPath: wasmPath,
					name:     pluginName,
					loader:   loader,
					loadFunc: func(ctx context.Context, l any, path string) (api.ArtistMetadataService, error) {
						return l.(*api.ArtistMetadataServicePlugin).Load(ctx, path)
					},
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
		createAdapter: func(loader any, wasmPath, pluginName string) any {
			return &wasmAlbumAgent{
				wasmBasePlugin: &wasmBasePlugin[api.AlbumMetadataService]{
					wasmPath: wasmPath,
					name:     pluginName,
					loader:   loader,
					loadFunc: func(ctx context.Context, l any, path string) (api.AlbumMetadataService, error) {
						return l.(*api.AlbumMetadataServicePlugin).Load(ctx, path)
					},
				},
			}
		},
	},
	"ScrobblerService": {
		loaderCtor: func(ctx context.Context, runtimeCtor func(context.Context) (wazero.Runtime, error), mc wazero.ModuleConfig) (any, error) {
			return api.NewScrobblerServicePlugin(ctx, api.WazeroRuntime(runtimeCtor), api.WazeroModuleConfig(mc))
		},
		loadFunc: func(ctx context.Context, loader any, wasmPath string) (any, error) {
			return loader.(*api.ScrobblerServicePlugin).Load(ctx, wasmPath)
		},
		createAdapter: func(loader any, wasmPath, pluginName string) any {
			return &wasmScrobblerPlugin{
				wasmBasePlugin: &wasmBasePlugin[api.ScrobblerService]{
					wasmPath: wasmPath,
					name:     pluginName,
					loader:   loader,
					loadFunc: func(ctx context.Context, l any, path string) (api.ScrobblerService, error) {
						return l.(*api.ScrobblerServicePlugin).Load(ctx, path)
					},
				},
			}
		},
	},
}

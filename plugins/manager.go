package plugins

//go:generate protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative api/api.proto
//go:generate protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative host/http.proto

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

var (
	compileSemaphore = make(chan struct{}, 2) // Limit to 2 concurrent compilations; adjust as needed
	compilationCache wazero.CompilationCache
	cacheOnce        sync.Once
)

func getCompilationCache() wazero.CompilationCache {
	cacheOnce.Do(func() {
		cacheDir := filepath.Join(conf.Server.CacheFolder, "plugins")
		var err error
		compilationCache, err = wazero.NewCompilationCacheWithDir(cacheDir)
		if err != nil {
			panic(fmt.Sprintf("Failed to create wazero compilation cache: %v", err))
		}
	})
	return compilationCache
}

type pluginState struct {
	ready chan struct{}
	err   error
}

// pooledInstance holds a wasm instance and its associated cleanup handle
type pooledInstance struct {
	instance any
	cleanup  runtime.Cleanup
}

// cleanupArg holds the necessary information for the GC cleanup function
type cleanupArg struct {
	closer     interface{ Close(context.Context) error }
	pluginName string
	wasmPath   string
}

// cleanupFunc is the function registered with runtime.AddCleanup
func cleanupFunc(arg cleanupArg) {
	log.Trace("pool: GC cleanup closing instance", "plugin", arg.pluginName, "path", arg.wasmPath)
	if err := arg.closer.Close(context.Background()); err != nil {
		log.Error("pool: GC cleanup failed to close instance", "plugin", arg.pluginName, "path", arg.wasmPath, err)
	} else {
		log.Trace("pool: GC cleanup closed instance successfully", "plugin", arg.pluginName, "path", arg.wasmPath)
	}
}

// Helper to create the correct ModuleConfig for plugins
func newWazeroModuleConfig() wazero.ModuleConfig {
	return wazero.NewModuleConfig().WithStartFunctions("_initialize").WithStderr(os.Stderr)
}

// Generic plugin pool creation
func newGenericPluginPool[L any](loader L, wasmPath, pluginName string, loadFunc func(L, context.Context, string) (any, error)) *sync.Pool {
	return &sync.Pool{
		New: func() any {
			inst, err := loadFunc(loader, context.Background(), wasmPath)
			if err != nil {
				log.Error("pool: failed to load plugin instance", "plugin", pluginName, "path", wasmPath, err)
				return nil
			}
			closer, ok := inst.(interface{ Close(context.Context) error })
			if !ok {
				return &pooledInstance{instance: inst}
			}
			arg := cleanupArg{
				closer:     closer,
				pluginName: pluginName,
				wasmPath:   wasmPath,
			}
			cleanup := runtime.AddCleanup(&inst, cleanupFunc, arg)
			return &pooledInstance{instance: inst, cleanup: cleanup}
		},
	}
}

// Manager is a singleton that manages plugins
type Manager struct{}

// GetManager returns the singleton instance of Manager
func GetManager() *Manager {
	return singleton.GetInstance(func() *Manager {
		return createManager()
	})
}

func createManager() *Manager {
	m := &Manager{}
	m.autoRegisterPlugins()
	return m
}

type pluginTypeInfo struct {
	loaderCtor func(context.Context, func(context.Context) (wazero.Runtime, error), wazero.ModuleConfig) (any, error)
	loadFunc   func(any, context.Context, string) (any, error)
	agentCtor  func(*sync.Pool, string, string) agents.Interface
}

var pluginTypes = map[string]pluginTypeInfo{
	"ArtistMetadataService": {
		loaderCtor: func(ctx context.Context, runtimeCtor func(context.Context) (wazero.Runtime, error), mc wazero.ModuleConfig) (any, error) {
			return api.NewArtistMetadataServicePlugin(ctx, api.WazeroRuntime(runtimeCtor), api.WazeroModuleConfig(mc))
		},
		loadFunc: func(loader any, ctx context.Context, wasmPath string) (any, error) {
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
		loadFunc: func(loader any, ctx context.Context, wasmPath string) (any, error) {
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

// In autoRegisterPlugins, create the loader once per plugin and pass to the pool
func (m *Manager) autoRegisterPlugins() {
	root := conf.Server.Plugins.Folder
	entries, err := os.ReadDir(root)
	if err != nil {
		log.Error("Failed to read plugins folder", "folder", root, err)
		return
	}
	cache := getCompilationCache()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		pluginDir := filepath.Join(root, name)
		wasmPath := filepath.Join(pluginDir, "plugin.wasm")
		if _, err := os.Stat(wasmPath); err != nil {
			log.Debug("No plugin.wasm found in plugin directory", "plugin", name, "path", wasmPath)
			continue
		}
		manifest, err := LoadManifest(pluginDir)
		if err != nil || len(manifest.Services) == 0 {
			log.Warn("No manifest or no services found in plugin directory", "plugin", name, "path", pluginDir, err)
			continue
		}
		for _, service := range manifest.Services {
			pt, ok := pluginTypes[service]
			if !ok {
				log.Warn("Unknown plugin service type in manifest", "service", service, "plugin", name)
				continue
			}
			customRuntime := func(ctx context.Context) (wazero.Runtime, error) {
				runtimeConfig := wazero.NewRuntimeConfig().WithCompilationCache(cache)
				r := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)
				if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
					return nil, err
				}
				if err := host.Instantiate(ctx, r, &HttpService{}); err != nil {
					return nil, err
				}
				return r, nil
			}
			mc := newWazeroModuleConfig()
			pluginName := name
			if len(manifest.Services) > 1 {
				pluginName = name + "_" + service
			}
			loaderAny, err := pt.loaderCtor(context.Background(), customRuntime, mc)
			if err != nil {
				log.Error("Failed to create plugin loader", "service", service, "plugin", name, err)
				continue
			}
			pool := newGenericPluginPool(loaderAny, wasmPath, pluginName, pt.loadFunc)
			agentFactory := func(ds model.DataStore) agents.Interface {
				return pt.agentCtor(pool, wasmPath, pluginName)
			}
			agents.Register(pluginName, agentFactory)
			log.Info("Registered plugin agent", "name", pluginName, "service", service, "wasm", wasmPath)
		}
	}
}

func init() {
	conf.AddHook(func() {
		GetManager()
	})
}

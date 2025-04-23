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
	"time"

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

// newWazeroRuntimeWithCache creates a custom wazero runtime with persistent cache
func newWazeroRuntimeWithCache() func(ctx context.Context) (wazero.Runtime, error) {
	cache := getCompilationCache()
	return func(ctx context.Context) (wazero.Runtime, error) {
		runtimeConfig := wazero.NewRuntimeConfig().WithCompilationCache(cache)
		r := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)
		if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
			return nil, err
		}
		return r, nil
	}
}

// Helper to create the correct ModuleConfig for plugins
func newWazeroModuleConfig() wazero.ModuleConfig {
	return wazero.NewModuleConfig().WithStartFunctions("_initialize").WithStderr(os.Stderr)
}

// newPluginPool creates and configures a sync.Pool for wasm plugin instances, using a pre-created loader.
func newPluginPool(pluginLoader *api.ArtistMetadataServicePlugin, wasmPath string, pluginName string) *sync.Pool {
	return &sync.Pool{
		New: func() any {
			inst, err := pluginLoader.Load(context.Background(), wasmPath)
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

// newAlbumPluginPool is analogous to newPluginPool but for AlbumMetadataService
func newAlbumPluginPool(pluginLoader *api.AlbumMetadataServicePlugin, wasmPath string, pluginName string) *sync.Pool {
	return &sync.Pool{
		New: func() any {
			inst, err := pluginLoader.Load(context.Background(), wasmPath)
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

// precompilePlugin compiles the wasm plugin in the background and updates the state.
func precompilePlugin(state *pluginState, wasmPath, name string) {
	compileSemaphore <- struct{}{}        // acquire slot
	defer func() { <-compileSemaphore }() // release slot
	ctx := context.Background()
	cache := getCompilationCache()
	b, err := os.ReadFile(wasmPath)
	if err != nil {
		state.err = fmt.Errorf("failed to read wasm file: %w", err)
		close(state.ready)
		return
	}
	runtimeConfig := wazero.NewRuntimeConfig().WithCompilationCache(cache)
	r := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)
	defer r.Close(ctx)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		state.err = fmt.Errorf("failed to instantiate WASI: %w", err)
		close(state.ready)
		return
	}
	start := time.Now()
	_, err = r.CompileModule(ctx, b)
	if err != nil {
		state.err = fmt.Errorf("failed to compile wasm: %w", err)
		log.Warn("Plugin compilation failed", "name", name, "path", wasmPath, "elapsed", time.Since(start), state.err)
	} else {
		state.err = nil
		log.Debug("Plugin compilation completed", "name", name, "path", wasmPath, "elapsed", time.Since(start))
	}
	close(state.ready)
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
			switch service {
			case "ArtistMetadataService":
				loader, err := api.NewArtistMetadataServicePlugin(context.Background(), api.WazeroRuntime(customRuntime), api.WazeroModuleConfig(mc))
				if err != nil {
					log.Error("Failed to create plugin loader", "service", service, "plugin", name, err)
					continue
				}
				pool := newPluginPool(loader, wasmPath, pluginName)
				agentFactory := func(ds model.DataStore) agents.Interface {
					return &wasmArtistAgent{
						wasmBasePlugin: &wasmBasePlugin[api.ArtistMetadataService]{
							pool:     pool,
							wasmPath: wasmPath,
							name:     pluginName,
						},
					}
				}
				agents.Register(pluginName, agentFactory)
			case "AlbumMetadataService":
				loader, err := api.NewAlbumMetadataServicePlugin(context.Background(), api.WazeroRuntime(customRuntime), api.WazeroModuleConfig(mc))
				if err != nil {
					log.Error("Failed to create plugin loader", "service", service, "plugin", name, err)
					continue
				}
				pool := newAlbumPluginPool(loader, wasmPath, pluginName)
				agentFactory := func(ds model.DataStore) agents.Interface {
					return &wasmAlbumAgent{
						wasmBasePlugin: &wasmBasePlugin[api.AlbumMetadataService]{
							pool:     pool,
							wasmPath: wasmPath,
							name:     pluginName,
						},
					}
				}
				agents.Register(pluginName, agentFactory)
			default:
				log.Warn("Unknown plugin service type in manifest", "service", service, "plugin", name)
			}
			log.Info("Registered plugin agent", "name", pluginName, "service", service, "wasm", wasmPath)
		}
	}
}

func init() {
	conf.AddHook(func() {
		GetManager()
	})
}

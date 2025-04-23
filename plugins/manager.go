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

// newPluginPool creates and configures a sync.Pool for wasm plugin instances.
func newPluginPool(pluginLoader *api.ArtistMetadataServicePlugin, wasmPath string, pluginName string) *sync.Pool {
	return &sync.Pool{
		New: func() any {
			inst, err := pluginLoader.Load(context.Background(), wasmPath)
			if err != nil {
				log.Error("pool: failed to load plugin instance", "plugin", pluginName, "path", wasmPath, err)
				return nil // Will cause getInstance to try again on next call
			}
			log.Trace("pool: created new plugin instance", "plugin", pluginName, "path", wasmPath)

			// Check if the instance has a Close(context.Context) error method
			closer, ok := inst.(interface{ Close(context.Context) error })
			if !ok {
				log.Trace("pool: instance does not implement Close(context.Context) error", "plugin", pluginName, "path", wasmPath)
				// Return instance without a cleanup handle (zero value for cleanup)
				return &pooledInstance{instance: inst}
			}

			arg := cleanupArg{
				closer:     closer,
				pluginName: pluginName,
				wasmPath:   wasmPath,
			}
			// Pass pointer &inst
			cleanup := runtime.AddCleanup(&inst, cleanupFunc, arg)
			log.Trace("pool: registered GC cleanup for instance", "plugin", pluginName, "path", wasmPath)

			return &pooledInstance{instance: inst, cleanup: cleanup}
		},
	}
}

// LoadAgentPlugin loads a WASM agent plugin and returns an implementation of agents.Interface and all retriever interfaces.
func LoadAgentPlugin(ctx context.Context, wasmPath string, name ...string) (agents.Interface, error) {
	// Setup persistent compilation cache
	_ = os.MkdirAll(filepath.Join(conf.Server.CacheFolder, "plugins"), 0o700)
	cache := getCompilationCache()
	customRuntime := func(ctx context.Context) (wazero.Runtime, error) {
		runtimeConfig := wazero.NewRuntimeConfig().WithCompilationCache(cache)
		r := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)
		// WASI imports
		if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
			log.Error(ctx, "Failed to instantiate WASI", err)
			return nil, err
		}
		return r, host.Instantiate(ctx, r, &HttpService{})
	}
	mc := wazero.NewModuleConfig().
		WithStartFunctions("_initialize").
		WithStderr(os.Stderr) // Redirect stderr to the host's stderr
	pluginLoader, err := api.NewArtistMetadataServicePlugin(ctx, api.WazeroRuntime(customRuntime), api.WazeroModuleConfig(mc))
	if err != nil {
		log.Error(ctx, "Failed to create plugin loader", "wasmPath", wasmPath, err)
		return nil, fmt.Errorf("failed to create plugin loader: %w", err)
	}
	pluginName := "wasm-plugin"
	if len(name) > 0 {
		pluginName = name[0]
	}
	// Create the pool using the helper function
	pool := newPluginPool(pluginLoader, wasmPath, pluginName)
	log.Trace(ctx, "Instantiated plugin agent", "plugin", pluginName, "path", wasmPath)
	return &wasmArtistAgent{
		pool:     pool,
		wasmPath: wasmPath,
		name:     pluginName,
	}, nil
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

// createAgentFactory returns a function suitable for agents.Register.
// This factory waits for pre-compilation and then loads the agent plugin.
func createAgentFactory(state *pluginState, wasmPath, name string) func(ds model.DataStore) agents.Interface {
	return func(ds model.DataStore) agents.Interface {
		<-state.ready
		if state.err != nil {
			log.Error("Failed to compile plugin", "name", name, "path", wasmPath, state.err)
			return nil
		}
		agent, err := LoadAgentPlugin(context.Background(), wasmPath, name)
		if err != nil {
			log.Error("Failed to load plugin", "name", name, "path", wasmPath, err)
			return nil
		}
		log.Debug("Loaded plugin agent", "name", name, "path", wasmPath)
		return agent
	}
}

// autoRegisterPlugins scans the plugins folder and registers each plugin found
func (m *Manager) autoRegisterPlugins() {
	root := conf.Server.Plugins.Folder
	entries, err := os.ReadDir(root)
	if err != nil {
		log.Error("Failed to read plugins folder", "folder", root, err)
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		wasmPath := filepath.Join(root, name, "plugin.wasm")
		if _, err := os.Stat(wasmPath); err != nil {
			log.Debug("No plugin.wasm found in plugin directory", "plugin", name, "path", wasmPath)
			continue
		}

		// Fix closure capture: copy variables
		localName := name
		localWasmPath := wasmPath
		state := &pluginState{ready: make(chan struct{})}

		// Start pre-compilation in the background
		go precompilePlugin(state, localWasmPath, localName)

		// Register the agent factory
		agents.Register(localName, createAgentFactory(state, localWasmPath, localName))

		log.Info("Registered plugin agent", "name", localName, "wasm", localWasmPath)
	}
}

func init() {
	conf.AddHook(func() {
		GetManager()
	})
}

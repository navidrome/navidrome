package plugins

//go:generate protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative api/api.proto
//go:generate protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative host/http.proto

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
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

// Helper to create the correct ModuleConfig for plugins
func newWazeroModuleConfig() wazero.ModuleConfig {
	return wazero.NewModuleConfig().WithStartFunctions("_initialize").WithStderr(os.Stderr)
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

// precompilePlugin compiles the WASM module in the background and updates the pluginState.
func precompilePlugin(state *pluginState, customRuntime func(context.Context) (wazero.Runtime, error), wasmPath, name string) {
	compileSemaphore <- struct{}{}
	defer func() { <-compileSemaphore }()
	ctx := context.Background()
	r, err := customRuntime(ctx)
	if err != nil {
		state.err = fmt.Errorf("failed to create runtime for plugin %s: %w", name, err)
		close(state.ready)
		return
	}
	defer r.Close(ctx)
	b, err := os.ReadFile(wasmPath)
	if err != nil {
		state.err = fmt.Errorf("failed to read wasm file: %w", err)
		close(state.ready)
		return
	}
	if _, err := r.CompileModule(ctx, b); err != nil {
		state.err = fmt.Errorf("failed to compile WASM for plugin %s: %w", name, err)
		log.Warn("Plugin compilation failed", "name", name, "path", wasmPath, "err", err)
	} else {
		state.err = nil
		log.Debug("Plugin compilation completed", "name", name, "path", wasmPath)
	}
	close(state.ready)
}

// createAgentFactory returns a factory that waits for precompilation before instantiating the agent.
func createAgentFactory(state *pluginState, loader any, wasmPath, pluginName string, agentCtor func(any, string, string) agents.Interface) func(model.DataStore) agents.Interface {
	return func(ds model.DataStore) agents.Interface {
		<-state.ready
		if state.err != nil {
			log.Error("Failed to compile plugin", "name", pluginName, "path", wasmPath, state.err)
			return nil
		}
		return agentCtor(loader, wasmPath, pluginName)
	}
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
			// Start pre-compilation in the background
			state := &pluginState{ready: make(chan struct{})}
			go precompilePlugin(state, customRuntime, wasmPath, pluginName)

			loaderAny, err := pt.loaderCtor(context.Background(), customRuntime, mc)
			if err != nil {
				log.Error("Failed to create plugin loader", "service", service, "plugin", name, err)
				continue
			}
			// Use createAgentFactory to wait for precompilation
			agentFactory := createAgentFactory(state, loaderAny, wasmPath, pluginName, pt.agentCtor)
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

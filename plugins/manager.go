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
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// LoadAgentPlugin loads a WASM agent plugin and returns an implementation of agents.Interface and all retriever interfaces.
func LoadAgentPlugin(ctx context.Context, wasmPath string, name ...string) (agents.Interface, error) {
	// Setup persistent compilation cache
	cacheDir := filepath.Join(conf.Server.CacheFolder, "plugins")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		log.Error(ctx, "Failed to create wazero cache dir", "dir", cacheDir, err)
		return nil, fmt.Errorf("failed to create wazero cache dir: %w", err)
	}
	cache, err := wazero.NewCompilationCacheWithDir(cacheDir)
	if err != nil {
		log.Error(ctx, "Failed to create wazero compilation cache", "dir", cacheDir, err)
		return nil, fmt.Errorf("failed to create wazero compilation cache: %w", err)
	}
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
	pool := &sync.Pool{
		New: func() any {
			inst, err := pluginLoader.Load(context.Background(), wasmPath)
			if err != nil {
				log.Error(nil, "pool: failed to load plugin instance", "plugin", pluginName, "path", wasmPath, err)
				return nil // Will cause getInstance to try again on next call
			}
			log.Trace(nil, "pool: created new plugin instance", "plugin", pluginName, "path", wasmPath)
			return inst
		},
	}
	log.Trace(ctx, "Instantiated plugin agent", "plugin", pluginName, "path", wasmPath)
	return &wasmAgent{
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

// autoRegisterPlugins scans the plugins folder and registers each plugin found
func (m *Manager) autoRegisterPlugins() {
	root := conf.Server.Plugins.Folder
	entries, err := os.ReadDir(root)
	if err != nil {
		log.Error(nil, "Failed to read plugins folder", "folder", root, err)
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		wasmPath := filepath.Join(root, name, "plugin.wasm")
		if _, err := os.Stat(wasmPath); err != nil {
			log.Debug(nil, "No plugin.wasm found in plugin directory", "plugin", name, "path", wasmPath)
			continue
		}
		agents.Register(name, func(ds model.DataStore) agents.Interface {
			agent, err := LoadAgentPlugin(context.Background(), wasmPath, name)
			if err != nil {
				log.Error(nil, "Failed to load plugin", "name", name, "path", wasmPath, err)
				return nil
			}
			log.Debug(nil, "Loaded plugin agent", "name", name, "path", wasmPath)
			return agent
		})
		log.Info(nil, "Registered plugin agent", "name", name, "wasm", wasmPath)
	}
}

func init() {
	conf.AddHook(func() {
		GetManager()
	})
}

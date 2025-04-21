package plugins

//go:generate protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative api/api.proto

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/utils/singleton"
	wazero "github.com/tetratelabs/wazero"
	wasi_snapshot_preview1 "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// LoadAgentPlugin loads a WASM agent plugin and returns an implementation of agents.Interface and all retriever interfaces.
func LoadAgentPlugin(ctx context.Context, wasmPath string, name ...string) (agents.Interface, error) {
	// Setup persistent compilation cache
	cacheDir := filepath.Join(conf.Server.CacheFolder, "plugins")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create wazero cache dir: %w", err)
	}
	cache, err := wazero.NewCompilationCacheWithDir(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create wazero compilation cache: %w", err)
	}
	// Ensure cache is closed on process exit (best effort)
	// (In production, you may want to manage this more globally)
	// defer cache.Close(ctx)

	customRuntime := func(ctx context.Context) (wazero.Runtime, error) {
		runtimeConfig := wazero.NewRuntimeConfig().WithCompilationCache(cache)
		r := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)
		// WASI imports
		if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
			return nil, err
		}
		return r, nil
	}
	plugin, err := api.NewArtistMetadataServicePlugin(ctx, api.WazeroRuntime(customRuntime))
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin loader: %w", err)
	}
	inst, err := plugin.Load(ctx, wasmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin: %w", err)
	}
	pluginName := "wasm-plugin"
	if len(name) > 0 {
		pluginName = name[0]
	}
	return &wasmAgent{inst: inst, name: pluginName}, nil
}

// Manager handles plugin discovery and registration
// Future logic for plugin auto-registration will go here
type Manager struct {
	// Add fields as needed for plugin state
}

// GetManager returns the singleton instance of Manager
func GetManager() *Manager {
	return singleton.GetInstance(func() *Manager {
		m := &Manager{}
		m.autoRegisterPlugins()
		return m
	})
}

// autoRegisterPlugins scans the plugins folder and registers each plugin found
func (m *Manager) autoRegisterPlugins() {
	root := conf.Server.Plugins.Folder
	entries, err := os.ReadDir(root)
	if err != nil {
		log.Error("Failed to read plugins folder", "folder", root, "err", err)
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		wasmPath := filepath.Join(root, name, "plugin.wasm")
		if _, err := os.Stat(wasmPath); err != nil {
			log.Debug("No plugin.wasm found in plugin directory", "plugin", name)
			continue
		}
		agents.Register(name, func(ds model.DataStore) agents.Interface {
			agent, err := LoadAgentPlugin(context.Background(), wasmPath, name)
			if err != nil {
				log.Error("Failed to load plugin", "name", name, "err", err)
				return nil
			}
			return agent
		})
		log.Info(nil, "Registered plugin agent", "name", name, "wasm", wasmPath)
	}
}

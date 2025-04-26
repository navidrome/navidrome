package plugins

import (
	"context"
	"sync"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/api"
)

// initializedPlugins tracks which plugins have been initialized
type initializedPlugins struct {
	plugins map[string]bool
	mu      sync.RWMutex
}

// newInitializedPlugins creates a new initialized plugins tracker
func newInitializedPlugins() *initializedPlugins {
	return &initializedPlugins{
		plugins: make(map[string]bool),
	}
}

// isInitialized checks if a plugin has been initialized
func (i *initializedPlugins) isInitialized(info *PluginInfo) bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.plugins[info.Name+consts.Zwsp+info.Manifest.Version]
}

// markInitialized marks a plugin as initialized
func (i *initializedPlugins) markInitialized(info *PluginInfo) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.plugins[info.Name+consts.Zwsp+info.Manifest.Version] = true
}

// callOnInit calls the OnInit method on a plugin that implements InitService
func (m *initializedPlugins) callOnInit(info *PluginInfo) {
	ctx := context.Background()
	log.Debug("Initializing plugin", "name", info.Name)
	start := time.Now()

	// Create InitService plugin instance
	loader, err := api.NewInitServicePlugin(ctx, api.WazeroRuntime(info.Runtime), api.WazeroModuleConfig(info.ModConfig))
	if loader == nil || err != nil {
		log.Error("Error creating InitService plugin", "plugin", info.Name, err)
		return
	}

	initPlugin, err := loader.Load(ctx, info.WasmPath)
	if err != nil {
		log.Error("Error loading InitService plugin", "plugin", info.Name, "path", info.WasmPath, err)
		return
	}
	defer initPlugin.Close(ctx)

	// Call OnInit
	resp, err := initPlugin.OnInit(ctx, &api.InitRequest{})
	if err != nil {
		log.Error("Error initializing plugin", "plugin", info.Name, "elapsed", time.Since(start), err)
		return
	}

	if resp.Error != "" {
		log.Error("Plugin reported error during initialization", "plugin", info.Name, resp.Error)
		return
	}

	log.Debug("Plugin initialized successfully", "plugin", info.Name, "elapsed", time.Since(start))
}

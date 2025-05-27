package plugins

import (
	"context"
	"maps"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/api"
)

// initializedPlugins tracks which plugins have been initialized
type initializedPlugins struct {
	plugins map[string]bool
	mu      sync.RWMutex
	confMu  sync.RWMutex // Mutex to protect configuration access
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

// callOnInit calls the OnInit method on a plugin that implements LifecycleManagement
func (i *initializedPlugins) callOnInit(info *PluginInfo) {
	ctx := context.Background()
	log.Debug("Initializing plugin", "name", info.Name)
	start := time.Now()

	// Create LifecycleManagement plugin instance
	loader, err := api.NewLifecycleManagementPlugin(ctx, api.WazeroRuntime(info.Runtime), api.WazeroModuleConfig(info.ModConfig))
	if loader == nil || err != nil {
		log.Error("Error creating LifecycleManagement plugin", "plugin", info.Name, err)
		return
	}

	initPlugin, err := loader.Load(ctx, info.WasmPath)
	if err != nil {
		log.Error("Error loading LifecycleManagement plugin", "plugin", info.Name, "path", info.WasmPath, err)
		return
	}
	defer initPlugin.Close(ctx)

	// Prepare the request with plugin-specific configuration
	req := &api.InitRequest{}

	// Add plugin configuration if available
	i.confMu.Lock()
	defer i.confMu.Unlock()
	if conf.Server.PluginConfig != nil {
		if pluginConfig, ok := conf.Server.PluginConfig[info.Name]; ok && len(pluginConfig) > 0 {
			req.Config = maps.Clone(pluginConfig)
			log.Debug("Passing configuration to plugin", "plugin", info.Name, "configKeys", len(pluginConfig))
		}
	}

	// Call OnInit
	resp, err := initPlugin.OnInit(ctx, req)
	if err != nil {
		log.Error("Error initializing plugin", "plugin", info.Name, "elapsed", time.Since(start), err)
		return
	}

	if resp.Error != "" {
		log.Error("Plugin reported error during initialization", "plugin", info.Name, "error", resp.Error)
		return
	}

	log.Debug("Plugin initialized successfully", "plugin", info.Name, "elapsed", time.Since(start))
}

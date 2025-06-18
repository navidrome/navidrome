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

// pluginLifecycleManager tracks which plugins have been initialized and manages their lifecycle
type pluginLifecycleManager struct {
	plugins map[string]bool
	mu      sync.RWMutex
	confMu  sync.RWMutex // Mutex to protect configuration access
}

// newPluginLifecycleManager creates a new plugin lifecycle manager
func newPluginLifecycleManager() *pluginLifecycleManager {
	return &pluginLifecycleManager{
		plugins: make(map[string]bool),
	}
}

// isInitialized checks if a plugin has been initialized
func (m *pluginLifecycleManager) isInitialized(info *pluginInfo) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.plugins[info.Name+consts.Zwsp+info.Manifest.Version]
}

// markInitialized marks a plugin as initialized
func (m *pluginLifecycleManager) markInitialized(info *pluginInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plugins[info.Name+consts.Zwsp+info.Manifest.Version] = true
}

// callOnInit calls the OnInit method on a plugin that implements LifecycleManagement
func (m *pluginLifecycleManager) callOnInit(info *pluginInfo) {
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
	m.confMu.Lock()
	defer m.confMu.Unlock()
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

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
	plugins sync.Map // string -> bool
	config  map[string]map[string]string
}

// newPluginLifecycleManager creates a new plugin lifecycle manager
func newPluginLifecycleManager() *pluginLifecycleManager {
	config := maps.Clone(conf.Server.PluginConfig)
	return &pluginLifecycleManager{
		config: config,
	}
}

// isInitialized checks if a plugin has been initialized
func (m *pluginLifecycleManager) isInitialized(plugin *plugin) bool {
	key := plugin.ID + consts.Zwsp + plugin.Manifest.Version
	value, exists := m.plugins.Load(key)
	return exists && value.(bool)
}

// markInitialized marks a plugin as initialized
func (m *pluginLifecycleManager) markInitialized(plugin *plugin) {
	key := plugin.ID + consts.Zwsp + plugin.Manifest.Version
	m.plugins.Store(key, true)
}

// callOnInit calls the OnInit method on a plugin that implements LifecycleManagement
func (m *pluginLifecycleManager) callOnInit(plugin *plugin) {
	ctx := context.Background()
	log.Debug("Initializing plugin", "name", plugin.ID)
	start := time.Now()

	// Create LifecycleManagement plugin instance
	loader, err := api.NewLifecycleManagementPlugin(ctx, api.WazeroRuntime(plugin.Runtime), api.WazeroModuleConfig(plugin.ModConfig))
	if loader == nil || err != nil {
		log.Error("Error creating LifecycleManagement plugin", "plugin", plugin.ID, err)
		return
	}

	initPlugin, err := loader.Load(ctx, plugin.WasmPath)
	if err != nil {
		log.Error("Error loading LifecycleManagement plugin", "plugin", plugin.ID, "path", plugin.WasmPath, err)
		return
	}
	defer initPlugin.Close(ctx)

	// Prepare the request with plugin-specific configuration
	req := &api.InitRequest{}

	// Add plugin configuration if available
	if m.config != nil {
		if pluginConfig, ok := m.config[plugin.ID]; ok && len(pluginConfig) > 0 {
			req.Config = maps.Clone(pluginConfig)
			log.Debug("Passing configuration to plugin", "plugin", plugin.ID, "configKeys", len(pluginConfig))
		}
	}

	// Call OnInit
	resp, err := initPlugin.OnInit(ctx, req)
	if err != nil {
		log.Error("Error initializing plugin", "plugin", plugin.ID, "elapsed", time.Since(start), err)
		return
	}

	if resp.Error != "" {
		log.Error("Plugin reported error during initialization", "plugin", plugin.ID, "error", resp.Error)
		return
	}

	log.Debug("Plugin initialized successfully", "plugin", plugin.ID, "elapsed", time.Since(start))
}

package plugins

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	extism "github.com/extism/go-sdk"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/tetratelabs/wazero"
)

const (
	// ManifestFunction is the name of the function that plugins must export
	// to provide their manifest.
	ManifestFunction = "nd_manifest"

	// DefaultTimeout is the default timeout for plugin function calls
	DefaultTimeout = 30 * time.Second
)

// Manager manages loading and lifecycle of WebAssembly plugins.
// It implements both agents.PluginLoader and scrobbler.PluginLoader interfaces.
type Manager struct {
	mu      sync.RWMutex
	plugins map[string]*pluginInstance
	ctx     context.Context
	cancel  context.CancelFunc
	cache   wazero.CompilationCache
}

// pluginInstance represents a loaded plugin
type pluginInstance struct {
	name     string // Plugin name (from filename)
	path     string // Path to the wasm file
	manifest *Manifest
	compiled *extism.CompiledPlugin
}

// GetManager returns a singleton instance of the plugin manager.
// The manager is not started automatically; call Start() to begin loading plugins.
func GetManager() *Manager {
	return singleton.GetInstance(func() *Manager {
		return &Manager{
			plugins: make(map[string]*pluginInstance),
		}
	})
}

// Start initializes the plugin manager and loads plugins from the configured folder.
// It should be called once during application startup when plugins are enabled.
func (m *Manager) Start(ctx context.Context) error {
	if !conf.Server.Plugins.Enabled {
		log.Debug("Plugin system is disabled")
		return nil
	}

	m.ctx, m.cancel = context.WithCancel(ctx)

	// Initialize wazero compilation cache for better performance
	m.cache = wazero.NewCompilationCache()

	folder := m.pluginsFolder()
	if folder == "" {
		log.Debug("No plugins folder configured")
		return nil
	}

	// Create plugins folder if it doesn't exist
	if err := os.MkdirAll(folder, 0755); err != nil {
		log.Error("Failed to create plugins folder", "folder", folder, err)
		return err
	}

	log.Info(ctx, "Starting plugin manager", "folder", folder)

	// Discover and load plugins
	if err := m.discoverPlugins(folder); err != nil {
		log.Error(ctx, "Error discovering plugins", err)
		return err
	}

	return nil
}

// Stop shuts down the plugin manager and releases all resources.
func (m *Manager) Stop() error {
	if m.cancel != nil {
		m.cancel()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Close all plugins
	for name, instance := range m.plugins {
		if instance.compiled != nil {
			if err := instance.compiled.Close(context.Background()); err != nil {
				log.Error("Error closing plugin", "plugin", name, err)
			}
		}
	}
	m.plugins = make(map[string]*pluginInstance)

	// Close compilation cache
	if m.cache != nil {
		if err := m.cache.Close(context.Background()); err != nil {
			log.Error("Error closing wazero cache", err)
		}
	}

	return nil
}

// PluginNames returns the names of all plugins that implement a particular capability.
// This is used by both agents and scrobbler systems to discover available plugins.
func (m *Manager) PluginNames(capability string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var names []string
	cap := Capability(capability)
	for name, instance := range m.plugins {
		if instance.manifest.HasCapability(cap) {
			names = append(names, name)
		}
	}
	return names
}

// LoadMediaAgent loads and returns a media agent plugin by name.
// Returns false if the plugin is not found or doesn't have the MetadataAgent capability.
func (m *Manager) LoadMediaAgent(name string) (agents.Interface, bool) {
	m.mu.RLock()
	instance, ok := m.plugins[name]
	m.mu.RUnlock()

	if !ok || !instance.manifest.HasCapability(CapabilityMetadataAgent) {
		return nil, false
	}

	// Create a new plugin instance for this agent
	agent, err := m.createMetadataAgent(instance)
	if err != nil {
		log.Error("Failed to create metadata agent from plugin", "plugin", name, err)
		return nil, false
	}

	return agent, true
}

// LoadScrobbler loads and returns a scrobbler plugin by name.
// Returns false if the plugin is not found or doesn't have the Scrobbler capability.
func (m *Manager) LoadScrobbler(name string) (scrobbler.Scrobbler, bool) {
	// Scrobbler capability is not yet implemented
	return nil, false
}

// PluginInfo contains basic information about a plugin for metrics/insights.
type PluginInfo struct {
	Name    string
	Version string
}

// GetPluginInfo returns information about all loaded plugins.
// This is used by the metrics/insights system.
func (m *Manager) GetPluginInfo() map[string]PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info := make(map[string]PluginInfo, len(m.plugins))
	for name, instance := range m.plugins {
		info[name] = PluginInfo{
			Name:    instance.manifest.Name,
			Version: instance.manifest.Version,
		}
	}
	return info
}

// pluginsFolder returns the configured plugins folder path
func (m *Manager) pluginsFolder() string {
	if conf.Server.Plugins.Folder != "" {
		return conf.Server.Plugins.Folder
	}
	// Default to DataFolder/plugins
	if conf.Server.DataFolder != "" {
		return filepath.Join(conf.Server.DataFolder, "plugins")
	}
	return ""
}

// discoverPlugins scans the plugins folder and loads all .wasm files
func (m *Manager) discoverPlugins(folder string) error {
	entries, err := os.ReadDir(folder)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug("Plugins folder does not exist", "folder", folder)
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".wasm") {
			continue
		}

		wasmPath := filepath.Join(folder, entry.Name())
		pluginName := strings.TrimSuffix(entry.Name(), ".wasm")

		if err := m.loadPlugin(pluginName, wasmPath); err != nil {
			log.Error(m.ctx, "Failed to load plugin", "plugin", pluginName, "path", wasmPath, err)
			continue
		}

		log.Info(m.ctx, "Loaded plugin", "plugin", pluginName, "manifest", m.plugins[pluginName].manifest.Name)
	}

	return nil
}

// loadPlugin loads a single plugin from a wasm file
func (m *Manager) loadPlugin(name, wasmPath string) error {
	// Read wasm file
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return err
	}

	// Get plugin-specific config from conf.Server.PluginConfig
	pluginConfig := m.getPluginConfig(name)

	// Create Extism manifest for this plugin
	// Note: We create a temporary plugin first to get the manifest,
	// then we'll create the final one with proper AllowedHosts
	tempManifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmData{
				Data: wasmBytes,
				Name: "main",
			},
		},
		Config: pluginConfig,
	}

	tempConfig := extism.PluginConfig{
		EnableWasi:    true,
		RuntimeConfig: wazero.NewRuntimeConfig().WithCompilationCache(m.cache),
	}

	// Create temporary plugin to read manifest
	tempPlugin, err := extism.NewPlugin(m.ctx, tempManifest, tempConfig, nil)
	if err != nil {
		return err
	}
	defer tempPlugin.Close(m.ctx)

	// Call nd_manifest to get plugin manifest
	exit, manifestBytes, err := tempPlugin.Call(ManifestFunction, nil)
	if err != nil {
		return err
	}
	if exit != 0 {
		return err
	}

	// Parse and validate manifest
	manifest, err := ParseManifest(manifestBytes)
	if err != nil {
		return err
	}
	if err := manifest.Validate(); err != nil {
		return err
	}

	// Now create the final compiled plugin with proper AllowedHosts
	finalManifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmData{
				Data: wasmBytes,
				Name: "main",
			},
		},
		Config:       pluginConfig,
		AllowedHosts: manifest.AllowedHosts(),
		Timeout:      uint64(DefaultTimeout.Milliseconds()),
	}

	finalConfig := extism.PluginConfig{
		EnableWasi:    true,
		RuntimeConfig: wazero.NewRuntimeConfig().WithCompilationCache(m.cache),
	}

	compiled, err := extism.NewCompiledPlugin(m.ctx, finalManifest, finalConfig, nil)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.plugins[name] = &pluginInstance{
		name:     name,
		path:     wasmPath,
		manifest: manifest,
		compiled: compiled,
	}
	m.mu.Unlock()

	return nil
}

// getPluginConfig returns the configuration for a specific plugin
func (m *Manager) getPluginConfig(name string) map[string]string {
	if conf.Server.PluginConfig == nil {
		return nil
	}
	return conf.Server.PluginConfig[name]
}

// createMetadataAgent creates a new MetadataAgent from a plugin instance
func (m *Manager) createMetadataAgent(instance *pluginInstance) (*MetadataAgent, error) {
	// Create a new plugin instance from the compiled plugin
	plugin, err := instance.compiled.Instance(m.ctx, extism.PluginInstanceConfig{
		ModuleConfig: wazero.NewModuleConfig().WithSysWalltime(),
	})
	if err != nil {
		return nil, err
	}

	return NewMetadataAgent(instance.name, plugin), nil
}

// Verify interface implementations at compile time
var (
	_ agents.PluginLoader    = (*Manager)(nil)
	_ scrobbler.PluginLoader = (*Manager)(nil)
)

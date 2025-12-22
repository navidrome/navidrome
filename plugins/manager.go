package plugins

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	extism "github.com/extism/go-sdk"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/rjeczalik/notify"
	"github.com/tetratelabs/wazero"
)

const (
	// manifestFunction is the name of the function that plugins must export
	// to provide their manifest.
	manifestFunction = "nd_manifest"

	// defaultTimeout is the default timeout for plugin function calls
	defaultTimeout = 30 * time.Second
)

// Manager manages loading and lifecycle of WebAssembly plugins.
// It implements both agents.PluginLoader and scrobbler.PluginLoader interfaces.
type Manager struct {
	mu      sync.RWMutex
	plugins map[string]*pluginInstance
	ctx     context.Context
	cancel  context.CancelFunc
	cache   wazero.CompilationCache
	stopped atomic.Bool    // Set to true when Stop() is called
	loadWg  sync.WaitGroup // Tracks in-flight plugin load operations

	// File watcher fields (used when AutoReload is enabled)
	watcherEvents  chan notify.EventInfo
	watcherDone    chan struct{}
	debounceTimers map[string]*time.Timer
	debounceMu     sync.Mutex
}

// pluginInstance represents a loaded plugin
type pluginInstance struct {
	name         string // Plugin name (from filename)
	path         string // Path to the wasm file
	manifest     *Manifest
	compiled     *extism.CompiledPlugin
	capabilities []Capability // Auto-detected capabilities based on exported functions
}

func (p *pluginInstance) create() (*extism.Plugin, error) {
	return p.compiled.Instance(context.Background(), extism.PluginInstanceConfig{
		ModuleConfig: wazero.NewModuleConfig().WithSysWalltime().WithRandSource(rand.Reader),
	})
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
		log.Debug(ctx, "Plugin system is disabled")
		return nil
	}

	m.ctx, m.cancel = context.WithCancel(ctx)

	// Initialize wazero compilation cache for better performance
	var err error
	m.cache, err = wazero.NewCompilationCacheWithDir(filepath.Join(conf.Server.CacheFolder, "plugins"))
	if err != nil {
		log.Error(ctx, "Failed to create wazero compilation cache", err)
		return fmt.Errorf("creating wazero compilation cache: %w", err)
	}

	folder := conf.Server.Plugins.Folder
	if folder == "" {
		log.Debug(ctx, "No plugins folder configured")
		return nil
	}

	// Create plugins folder if it doesn't exist
	if err := os.MkdirAll(folder, 0755); err != nil {
		log.Error(ctx, "Failed to create plugins folder", "folder", folder, err)
		return fmt.Errorf("creating plugins folder: %w", err)
	}

	log.Info(ctx, "Starting plugin manager", "folder", folder)

	// Discover and load plugins
	if err := m.discoverPlugins(folder); err != nil {
		log.Error(ctx, "Error discovering plugins", err)
		return fmt.Errorf("discovering plugins: %w", err)
	}

	// Start file watcher if auto-reload is enabled
	if conf.Server.Plugins.AutoReload {
		if err := m.startWatcher(); err != nil {
			log.Error(ctx, "Failed to start plugin file watcher", err)
			// Non-fatal - plugins are still loaded, just no auto-reload
		}
	}

	return nil
}

// Stop shuts down the plugin manager and releases all resources.
func (m *Manager) Stop() error {
	// Mark as stopped first to prevent new operations
	m.stopped.Store(true)

	// Cancel context to signal all goroutines to stop
	if m.cancel != nil {
		m.cancel()
	}

	// Stop file watcher
	m.stopWatcher()

	// Wait for all in-flight plugin load operations to complete
	// This is critical to avoid races with cache.Close()
	m.loadWg.Wait()

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
		m.cache = nil
	}

	return nil
}

// PluginNames returns the names of all plugins that implement a particular capability.
// This is used by both agents and scrobbler systems to discover available plugins.
// Capabilities are auto-detected from the plugin's exported functions.
func (m *Manager) PluginNames(capability string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var names []string
	cap := Capability(capability)
	for name, instance := range m.plugins {
		if hasCapability(instance.capabilities, cap) {
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

	if !ok || !hasCapability(instance.capabilities, CapabilityMetadataAgent) {
		return nil, false
	}

	// Create a new metadata agent adapter for this plugin
	return &MetadataAgent{
		name:   instance.name,
		plugin: instance,
	}, true
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
	// Check if manager is stopped (cache may be closed)
	if m.stopped.Load() {
		return fmt.Errorf("manager is stopped")
	}

	// Track this operation so Stop() can wait for it to complete
	m.loadWg.Add(1)
	defer m.loadWg.Done()

	// Double-check after adding to WaitGroup (Stop may have been called between check and Add)
	if m.stopped.Load() {
		return fmt.Errorf("manager is stopped")
	}

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

	// Create temporary plugin to read manifest and detect capabilities
	tempPlugin, err := extism.NewPlugin(m.ctx, tempManifest, tempConfig, nil)
	if err != nil {
		return err
	}
	defer tempPlugin.Close(m.ctx)

	// Call nd_manifest to get plugin manifest
	exit, manifestBytes, err := tempPlugin.Call(manifestFunction, nil)
	if err != nil {
		return err
	}
	if exit != 0 {
		return fmt.Errorf("calling %s: %d", manifestFunction, exit)
	}

	// Parse manifest (validation happens during unmarshal via generated code)
	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("invalid plugin manifest: %w", err)
	}

	// Detect capabilities based on exported functions
	capabilities := detectCapabilities(tempPlugin)

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
		Timeout:      uint64(defaultTimeout.Milliseconds()),
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
		name:         name,
		path:         wasmPath,
		manifest:     &manifest,
		compiled:     compiled,
		capabilities: capabilities,
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

// UnloadPlugin removes a plugin from the manager and closes its resources.
// Returns an error if the plugin is not found.
func (m *Manager) UnloadPlugin(name string) error {
	m.mu.Lock()
	instance, ok := m.plugins[name]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("plugin %q not found", name)
	}
	delete(m.plugins, name)
	m.mu.Unlock()

	// Close the compiled plugin outside the lock with a grace period
	// to allow in-flight requests to complete
	if instance.compiled != nil {
		// Use a brief timeout for cleanup
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := instance.compiled.Close(ctx); err != nil {
			log.Error("Error closing plugin during unload", "plugin", name, err)
		}
	}

	log.Info(m.ctx, "Unloaded plugin", "plugin", name)
	return nil
}

// LoadPlugin loads a new plugin by name from the plugins folder.
// The plugin file must exist at <plugins_folder>/<name>.wasm.
// Returns an error if the plugin is already loaded or fails to load.
func (m *Manager) LoadPlugin(name string) error {
	m.mu.RLock()
	_, exists := m.plugins[name]
	m.mu.RUnlock()

	if exists {
		return fmt.Errorf("plugin %q is already loaded", name)
	}

	folder := conf.Server.Plugins.Folder
	if folder == "" {
		return fmt.Errorf("no plugins folder configured")
	}

	wasmPath := filepath.Join(folder, name+".wasm")
	if _, err := os.Stat(wasmPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("plugin file not found: %s", wasmPath)
		}
		return err
	}

	if err := m.loadPlugin(name, wasmPath); err != nil {
		return fmt.Errorf("failed to load plugin %q: %w", name, err)
	}

	log.Info(m.ctx, "Loaded plugin", "plugin", name)
	return nil
}

// ReloadPlugin unloads and reloads a plugin by name.
// If the plugin was loaded and unload succeeds but reload fails,
// the plugin remains unloaded and the error is returned.
func (m *Manager) ReloadPlugin(name string) error {
	if err := m.UnloadPlugin(name); err != nil {
		return fmt.Errorf("failed to unload plugin %q: %w", name, err)
	}

	if err := m.LoadPlugin(name); err != nil {
		log.Error(m.ctx, "Failed to reload plugin, plugin remains unloaded", "plugin", name, err)
		return fmt.Errorf("failed to reload plugin %q: %w", name, err)
	}

	log.Info(m.ctx, "Reloaded plugin", "plugin", name)
	return nil
}

// callPluginFunction is a helper to call a plugin function with input and output types.
// It handles JSON marshalling/unmarshalling and error checking.
func callPluginFunction[I any, O any](ctx context.Context, plugin *pluginInstance, funcName string, input I) (O, error) {
	start := time.Now()

	var result O

	// Create plugin instance
	p, err := plugin.create()
	if err != nil {
		return result, fmt.Errorf("failed to create plugin: %w", err)
	}
	defer p.Close(ctx)

	if !p.FunctionExists(funcName) {
		return result, fmt.Errorf("%s does not exist", funcName)
	}

	inputBytes, err := json.Marshal(input)
	if err != nil {
		return result, fmt.Errorf("failed to marshal input: %w", err)
	}

	startCall := time.Now()
	exit, output, err := p.Call(funcName, inputBytes)
	if err != nil {
		log.Trace(ctx, "Plugin call failed", "p", plugin.name, "function", funcName, "pluginDuration", time.Since(startCall), "navidromeDuration", startCall.Sub(start), err)
		return result, fmt.Errorf("plugin call failed: %w", err)
	}
	if exit != 0 {
		return result, fmt.Errorf("plugin call exited with code %d", exit)
	}

	err = json.Unmarshal(output, &result)
	if err != nil {
		log.Trace(ctx, "Plugin call failed", "p", plugin.name, "function", funcName, "pluginDuration", time.Since(startCall), "navidromeDuration", startCall.Sub(start), err)
	}

	log.Trace(ctx, "Plugin call succeeded", "p", plugin.name, "function", funcName, "pluginDuration", time.Since(startCall), "navidromeDuration", startCall.Sub(start))
	return result, err
}

// Verify interface implementations at compile time
var (
	_ agents.PluginLoader    = (*Manager)(nil)
	_ scrobbler.PluginLoader = (*Manager)(nil)
)

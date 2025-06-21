package plugins

//go:generate protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative api/api.proto
//go:generate protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative host/http/http.proto
//go:generate protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative host/config/config.proto
//go:generate protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative host/websocket/websocket.proto
//go:generate protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative host/scheduler/scheduler.proto
//go:generate protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative host/cache/cache.proto
//go:generate protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative host/artwork/artwork.proto

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/schema"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/tetratelabs/wazero"
)

const (
	CapabilityMetadataAgent       = "MetadataAgent"
	CapabilityScrobbler           = "Scrobbler"
	CapabilitySchedulerCallback   = "SchedulerCallback"
	CapabilityWebSocketCallback   = "WebSocketCallback"
	CapabilityLifecycleManagement = "LifecycleManagement"
)

// pluginCreators maps capability types to their respective creator functions
type pluginConstructor func(wasmPath, pluginID string, runtime api.WazeroNewRuntime, mc wazero.ModuleConfig) WasmPlugin

var pluginCreators = map[string]pluginConstructor{
	CapabilityMetadataAgent:     newWasmMediaAgent,
	CapabilityScrobbler:         newWasmScrobblerPlugin,
	CapabilitySchedulerCallback: newWasmSchedulerCallback,
	CapabilityWebSocketCallback: newWasmWebSocketCallback,
}

// WasmPlugin is the base interface that all WASM plugins implement
type WasmPlugin interface {
	// PluginID returns the unique identifier of the plugin (folder name)
	PluginID() string
	// Instantiate creates a new instance of the plugin and returns it along with a cleanup function
	Instantiate(ctx context.Context) (any, func(), error)
}

type plugin struct {
	ID               string
	Path             string
	Capabilities     []string
	WasmPath         string
	Manifest         *schema.PluginManifest // Loaded manifest
	Runtime          api.WazeroNewRuntime
	ModConfig        wazero.ModuleConfig
	compilationReady chan struct{}
	compilationErr   error
}

func (p *plugin) waitForCompilation() error {
	timeout := pluginCompilationTimeout()
	select {
	case <-p.compilationReady:
	case <-time.After(timeout):
		err := fmt.Errorf("timed out waiting for plugin %s to compile", p.ID)
		log.Error("Timed out waiting for plugin compilation", "name", p.ID, "path", p.WasmPath, "timeout", timeout, "err", err)
		return err
	}
	if p.compilationErr != nil {
		log.Error("Failed to compile plugin", "name", p.ID, "path", p.WasmPath, p.compilationErr)
	}
	return p.compilationErr
}

// Manager is a singleton that manages plugins
type Manager struct {
	plugins          map[string]*plugin      // Map of plugin folder name to plugin info
	mu               sync.RWMutex            // Protects plugins map
	schedulerService *schedulerService       // Service for handling scheduled tasks
	websocketService *websocketService       // Service for handling WebSocket connections
	lifecycle        *pluginLifecycleManager // Manages plugin lifecycle and initialization
	adapters         map[string]WasmPlugin   // Map of plugin folder name + capability to adapter
}

// GetManager returns the singleton instance of Manager
func GetManager() *Manager {
	return singleton.GetInstance(func() *Manager {
		return createManager()
	})
}

// createManager creates a new Manager instance. Used in tests
func createManager() *Manager {
	m := &Manager{
		plugins:   make(map[string]*plugin),
		lifecycle: newPluginLifecycleManager(),
	}

	// Create the host services
	m.schedulerService = newSchedulerService(m)
	m.websocketService = newWebsocketService(m)

	return m
}

// registerPlugin adds a plugin to the registry with the given parameters
// Used internally by ScanPlugins to register plugins
func (m *Manager) registerPlugin(pluginID, pluginDir, wasmPath string, manifest *schema.PluginManifest) *plugin {
	// Create custom runtime function
	customRuntime := m.createRuntime(pluginID, manifest.Permissions)

	// Configure module and determine plugin name
	mc := newWazeroModuleConfig()

	// Check if it's a symlink, indicating development mode
	isSymlink := false
	if fileInfo, err := os.Lstat(pluginDir); err == nil {
		isSymlink = fileInfo.Mode()&os.ModeSymlink != 0
	}

	// Store plugin info
	p := &plugin{
		ID:               pluginID,
		Path:             pluginDir,
		Capabilities:     slice.Map(manifest.Capabilities, func(cap schema.PluginManifestCapabilitiesElem) string { return string(cap) }),
		WasmPath:         wasmPath,
		Manifest:         manifest,
		Runtime:          customRuntime,
		ModConfig:        mc,
		compilationReady: make(chan struct{}),
	}

	// Start pre-compilation of WASM module in background
	go func() {
		precompilePlugin(p)
		// Check if this plugin implements InitService and hasn't been initialized yet
		m.initializePluginIfNeeded(p)
	}()

	// Register the plugin
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plugins[pluginID] = p

	// Register one plugin adapter for each capability
	for _, capability := range manifest.Capabilities {
		capabilityStr := string(capability)
		constructor := pluginCreators[capabilityStr]
		if constructor == nil {
			// Warn about unknown capabilities, except for LifecycleManagement (it does not have an adapter)
			if capability != CapabilityLifecycleManagement {
				log.Warn("Unknown plugin capability type", "capability", capability, "plugin", pluginID)
			}
			continue
		}
		adapter := constructor(wasmPath, pluginID, customRuntime, mc)
		m.adapters[pluginID+"_"+capabilityStr] = adapter
	}

	log.Info("Discovered plugin", "folder", pluginID, "name", manifest.Name, "capabilities", manifest.Capabilities, "wasm", wasmPath, "dev_mode", isSymlink)
	return m.plugins[pluginID]
}

// initializePluginIfNeeded calls OnInit on plugins that implement LifecycleManagement
func (m *Manager) initializePluginIfNeeded(plugin *plugin) {
	// Skip if already initialized
	if m.lifecycle.isInitialized(plugin) {
		return
	}

	// Check if the plugin implements LifecycleManagement
	for _, capability := range plugin.Manifest.Capabilities {
		if capability == CapabilityLifecycleManagement {
			m.lifecycle.callOnInit(plugin)
			m.lifecycle.markInitialized(plugin)
			break
		}
	}
}

// ScanPlugins scans the plugins directory, discovers all valid plugins, and registers them for use.
func (m *Manager) ScanPlugins() {
	// Clear existing plugins
	m.mu.Lock()
	m.plugins = make(map[string]*plugin)
	m.adapters = make(map[string]WasmPlugin)
	m.mu.Unlock()

	// Get plugins directory from config
	root := conf.Server.Plugins.Folder
	log.Debug("Scanning plugins folder", "root", root)

	// Fail fast if the compilation cache cannot be initialized
	_, err := getCompilationCache()
	if err != nil {
		log.Error("Failed to initialize plugins compilation cache. Disabling plugins", err)
		return
	}

	// Discover all plugins using the shared discovery function
	discoveries := DiscoverPlugins(root)

	var validPluginNames []string
	for _, discovery := range discoveries {
		if discovery.Error != nil {
			// Handle global errors (like directory read failure)
			if discovery.ID == "" {
				log.Error("Plugin discovery failed", discovery.Error)
				return
			}
			// Handle individual plugin errors
			log.Error("Failed to process plugin", "plugin", discovery.ID, discovery.Error)
			continue
		}

		// Log discovery details
		log.Debug("Processing entry", "name", discovery.ID, "isSymlink", discovery.IsSymlink)
		if discovery.IsSymlink {
			log.Debug("Processing symlinked plugin directory", "name", discovery.ID, "target", discovery.Path)
		}
		log.Debug("Checking for plugin.wasm", "wasmPath", discovery.WasmPath)
		log.Debug("Manifest loaded successfully", "folder", discovery.ID, "name", discovery.Manifest.Name, "capabilities", discovery.Manifest.Capabilities)

		validPluginNames = append(validPluginNames, discovery.ID)

		// Register the plugin
		m.registerPlugin(discovery.ID, discovery.Path, discovery.WasmPath, discovery.Manifest)
	}

	log.Debug("Found valid plugins", "count", len(validPluginNames), "plugins", validPluginNames)
}

// PluginNames returns the folder names of all plugins that implement the specified capability
func (m *Manager) PluginNames(capability string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var names []string
	for name, plugin := range m.plugins {
		for _, c := range plugin.Manifest.Capabilities {
			if string(c) == capability {
				names = append(names, name)
				break
			}
		}
	}
	return names
}

func (m *Manager) getPlugin(name string, capability string) (*plugin, WasmPlugin) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	info, infoOk := m.plugins[name]
	adapter, adapterOk := m.adapters[name+"_"+capability]

	if !infoOk {
		log.Warn("Plugin not found", "name", name)
		return nil, nil
	}
	if !adapterOk {
		log.Warn("Plugin adapter not found", "name", name, "capability", capability)
		return nil, nil
	}
	return info, adapter
}

// LoadPlugin instantiates and returns a plugin by folder name
func (m *Manager) LoadPlugin(name string, capability string) WasmPlugin {
	info, adapter := m.getPlugin(name, capability)
	if info == nil {
		log.Warn("Plugin not found", "name", name, "capability", capability)
		return nil
	}

	log.Debug("Loading plugin", "name", name, "path", info.Path)

	// Wait for the plugin to be ready before using it.
	if err := info.waitForCompilation(); err != nil {
		log.Error("Plugin is not ready, cannot be loaded", "plugin", name, "capability", capability, "err", err)
		return nil
	}

	if adapter == nil {
		log.Warn("Plugin adapter not found", "name", name, "capability", capability)
		return nil
	}
	return adapter
}

// EnsureCompiled waits for a plugin to finish compilation and returns any compilation error.
// This is useful when you need to wait for compilation without loading a specific capability,
// such as during plugin refresh operations or health checks.
func (m *Manager) EnsureCompiled(name string) error {
	m.mu.RLock()
	plugin, ok := m.plugins[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("plugin not found: %s", name)
	}

	return plugin.waitForCompilation()
}

// LoadAllPlugins instantiates and returns all plugins that implement the specified capability
func (m *Manager) LoadAllPlugins(capability string) []WasmPlugin {
	names := m.PluginNames(capability)
	if len(names) == 0 {
		return nil
	}

	var plugins []WasmPlugin
	for _, name := range names {
		plugin := m.LoadPlugin(name, capability)
		if plugin != nil {
			plugins = append(plugins, plugin)
		}
	}
	return plugins
}

// LoadMediaAgent instantiates and returns a media agent plugin by folder name
func (m *Manager) LoadMediaAgent(name string) (agents.Interface, bool) {
	plugin := m.LoadPlugin(name, CapabilityMetadataAgent)
	if plugin == nil {
		return nil, false
	}
	agent, ok := plugin.(*wasmMediaAgent)
	return agent, ok
}

// LoadAllMediaAgents instantiates and returns all media agent plugins
func (m *Manager) LoadAllMediaAgents() []agents.Interface {
	plugins := m.LoadAllPlugins(CapabilityMetadataAgent)

	return slice.Map(plugins, func(p WasmPlugin) agents.Interface {
		return p.(agents.Interface)
	})
}

// LoadScrobbler instantiates and returns a scrobbler plugin by folder name
func (m *Manager) LoadScrobbler(name string) (scrobbler.Scrobbler, bool) {
	plugin := m.LoadPlugin(name, CapabilityScrobbler)
	if plugin == nil {
		return nil, false
	}
	s, ok := plugin.(scrobbler.Scrobbler)
	return s, ok
}

// LoadAllScrobblers instantiates and returns all scrobbler plugins
func (m *Manager) LoadAllScrobblers() []scrobbler.Scrobbler {
	plugins := m.LoadAllPlugins(CapabilityScrobbler)

	return slice.Map(plugins, func(p WasmPlugin) scrobbler.Scrobbler {
		return p.(scrobbler.Scrobbler)
	})
}

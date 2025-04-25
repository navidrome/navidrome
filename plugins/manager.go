package plugins

//go:generate protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative api/api.proto
//go:generate protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative host/host.proto

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// pluginCreators maps service types to their respective creator functions
type pluginConstructor func(wasmPath, pluginName string, runtime api.WazeroNewRuntime, mc wazero.ModuleConfig) WasmPlugin

var pluginCreators = map[string]pluginConstructor{
	"MediaMetadataService": NewWasmMediaAgent,
	"ScrobblerService":     NewWasmScrobblerPlugin,
}

// WasmPlugin is the base interface that all WASM plugins implement
type WasmPlugin interface {
	// PluginName returns the name of the plugin
	PluginName() string
}

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

// PluginInfo represents a plugin that has been discovered but not yet instantiated
type PluginInfo struct {
	Name      string
	Path      string
	Services  []string
	WasmPath  string
	Manifest  *PluginManifest
	State     *pluginState
	Runtime   api.WazeroNewRuntime
	ModConfig wazero.ModuleConfig
}

// Manager is a singleton that manages plugins
type Manager struct {
	plugins map[string]*PluginInfo // Map of plugin name to plugin info
	mu      sync.RWMutex           // Protects plugins map
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
		plugins: make(map[string]*PluginInfo),
	}
	return m
}

// precompilePlugin compiles the WASM module in the background and updates the pluginState.
func precompilePlugin(state *pluginState, customRuntime api.WazeroNewRuntime, wasmPath, name string) {
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

func pluginCompilationTimeout() time.Duration {
	if conf.Server.DevPluginCompilationTimeout > 0 {
		return conf.Server.DevPluginCompilationTimeout
	}
	return time.Minute
}

// waitForPluginReady blocks until the plugin is compiled and returns true if ready, false otherwise.
func waitForPluginReady(state *pluginState, pluginName, wasmPath string) bool {
	timeout := pluginCompilationTimeout()
	select {
	case <-state.ready:
	case <-time.After(timeout):
		log.Error("Timed out waiting for plugin compilation", "name", pluginName, "path", wasmPath, "timeout", timeout)
		return false
	}
	if state.err != nil {
		log.Error("Failed to compile plugin", "name", pluginName, "path", wasmPath, state.err)
		return false
	}
	return true
}

// ScanPlugins scans the plugins directory and compiles all valid plugins without registering them.
func (m *Manager) ScanPlugins() {
	// Get plugins directory from config and read its contents
	root := conf.Server.Plugins.Folder
	entries, err := os.ReadDir(root)
	if err != nil {
		log.Error("Failed to read plugins folder", "folder", root, err)
		return
	}
	// Get compilation cache to speed up WASM module loading
	cache := getCompilationCache()

	// Clear existing plugins
	m.mu.Lock()
	m.plugins = make(map[string]*PluginInfo)
	m.mu.Unlock()

	// Process each directory in the plugins folder
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		pluginDir := filepath.Join(root, name)
		wasmPath := filepath.Join(pluginDir, "plugin.wasm")

		// Skip if no WASM file found
		if _, err := os.Stat(wasmPath); err != nil {
			log.Debug("No plugin.wasm found in plugin directory", "plugin", name, "path", wasmPath)
			continue
		}

		// Load and validate plugin manifest
		manifest, err := LoadManifest(pluginDir)
		if err != nil || len(manifest.Services) == 0 {
			log.Warn("No manifest or no services found in plugin directory", "plugin", name, "path", pluginDir, err)
			continue
		}

		// Process each service defined in the manifest
		for _, service := range manifest.Services {
			_, ok := pluginCreators[service]
			if !ok {
				log.Warn("Unknown plugin service type in manifest", "service", service, "plugin", name)
				continue
			}

			// Create a custom WASM runtime with caching and required host functions
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

			// Configure module and determine plugin name
			mc := newWazeroModuleConfig()
			pluginName := name
			if len(manifest.Services) > 1 {
				pluginName = name + "_" + service
			}

			// Start pre-compilation of WASM module in background
			state := &pluginState{ready: make(chan struct{})}
			go precompilePlugin(state, customRuntime, wasmPath, pluginName)

			// Store plugin info
			m.mu.Lock()
			m.plugins[pluginName] = &PluginInfo{
				Name:      pluginName,
				Path:      pluginDir,
				Services:  []string{service},
				WasmPath:  wasmPath,
				Manifest:  manifest,
				State:     state,
				Runtime:   customRuntime,
				ModConfig: mc,
			}
			m.mu.Unlock()

			log.Info("Discovered plugin", "name", pluginName, "service", service, "wasm", wasmPath)
		}
	}
}

// PluginNames returns the names of all plugins that implement the specified service
func (m *Manager) PluginNames(svcName string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var names []string
	for name, plugin := range m.plugins {
		for _, svc := range plugin.Services {
			if svc == svcName {
				names = append(names, name)
				break
			}
		}
	}
	return names
}

// LoadPlugin instantiates and returns a plugin by name
func (m *Manager) LoadPlugin(name string) WasmPlugin {
	m.mu.RLock()
	plugin, ok := m.plugins[name]
	m.mu.RUnlock()

	if !ok {
		log.Warn("Plugin not found", "name", name)
		return nil
	}

	log.Debug("Loading plugin", "name", name, "path", plugin.WasmPath)

	if !waitForPluginReady(plugin.State, plugin.Name, plugin.WasmPath) {
		log.Warn("Plugin not ready", "name", name)
		return nil
	}

	if len(plugin.Services) == 0 {
		log.Warn("Plugin has no services", "name", name)
		return nil
	}

	serviceType := plugin.Services[0]
	creator, ok := pluginCreators[serviceType]
	if !ok {
		log.Warn("Unknown plugin service type", "service", serviceType, "plugin", name)
		return nil
	}

	// Use the creator based on the service type
	adapter := creator(plugin.WasmPath, plugin.Name, plugin.Runtime, plugin.ModConfig)

	if adapter == nil {
		log.Warn("Failed to create adapter for plugin", "name", name)
	}
	return adapter
}

// LoadMediaAgent instantiates and returns a media agent plugin by name
func (m *Manager) LoadMediaAgent(name string) (agents.Interface, bool) {
	plugin := m.LoadPlugin(name)
	if plugin == nil {
		return nil, false
	}
	agent, ok := plugin.(*wasmMediaAgent)
	return agent, ok
}

// LoadScrobbler instantiates and returns a scrobbler plugin by name
func (m *Manager) LoadScrobbler(name string) (scrobbler.Scrobbler, bool) {
	plugin := m.LoadPlugin(name)
	if plugin == nil {
		return nil, false
	}
	s, ok := plugin.(scrobbler.Scrobbler)
	return s, ok
}

// LoadAllPlugins instantiates and returns all plugins that implement the specified service
func (m *Manager) LoadAllPlugins(svcName string) []WasmPlugin {
	names := m.PluginNames(svcName)
	if len(names) == 0 {
		return nil
	}

	var plugins []WasmPlugin
	for _, name := range names {
		plugin := m.LoadPlugin(name)
		if plugin != nil {
			plugins = append(plugins, plugin)
		}
	}
	return plugins
}

// LoadAllMediaAgents instantiates and returns all media agent plugins
func (m *Manager) LoadAllMediaAgents() []agents.Interface {
	names := m.PluginNames("MediaMetadataService")
	if len(names) == 0 {
		return nil
	}

	var agents []agents.Interface
	for _, name := range names {
		agent, ok := m.LoadMediaAgent(name)
		if ok {
			agents = append(agents, agent)
		}
	}
	return agents
}

// LoadAllScrobblers instantiates and returns all scrobbler plugins
func (m *Manager) LoadAllScrobblers() []scrobbler.Scrobbler {
	names := m.PluginNames("ScrobblerService")
	if len(names) == 0 {
		return nil
	}

	var scrobblers []scrobbler.Scrobbler
	for _, name := range names {
		scrobbler, ok := m.LoadScrobbler(name)
		if ok {
			scrobblers = append(scrobblers, scrobbler)
		}
	}
	return scrobblers
}

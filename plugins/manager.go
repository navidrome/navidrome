package plugins

//go:generate protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative api/api.proto
//go:generate protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative host/http/http.proto
//go:generate protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative host/timer/timer.proto

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/http"
	"github.com/navidrome/navidrome/plugins/host/timer"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/tetratelabs/wazero"
	wazeroapi "github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const (
	CapabilityMetadataAgent = "MetadataAgent"
	CapabilityScrobbler     = "Scrobbler"
	CapabilityTimerCallback = "TimerCallback"
)

// pluginCreators maps capability types to their respective creator functions
type pluginConstructor func(wasmPath, pluginName string, runtime api.WazeroNewRuntime, mc wazero.ModuleConfig) WasmPlugin

var pluginCreators = map[string]pluginConstructor{
	CapabilityMetadataAgent: NewWasmMediaAgent,
	CapabilityScrobbler:     NewWasmScrobblerPlugin,
	CapabilityTimerCallback: NewWasmTimerCallback,
}

// WasmPlugin is the base interface that all WASM plugins implement
type WasmPlugin interface {
	// PluginName returns the name of the plugin
	PluginName() string
	ServiceType() string
	GetInstance(ctx context.Context) (any, func(), error)
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
	return wazero.NewModuleConfig().WithStartFunctions("_initialize").WithStderr(log.Writer())
}

// PluginInfo represents a plugin that has been discovered but not yet instantiated
type PluginInfo struct {
	Name         string
	Path         string
	Capabilities []string
	WasmPath     string
	Manifest     *PluginManifest
	State        *pluginState
	Runtime      api.WazeroNewRuntime
	ModConfig    wazero.ModuleConfig
}

// Manager is a singleton that manages plugins
type Manager struct {
	plugins      map[string]*PluginInfo // Map of plugin name to plugin info
	mu           sync.RWMutex           // Protects plugins map
	timerService *TimerService          // Service for handling plugin timers
	initialized  *initializedPlugins    // Tracks which plugins have been initialized
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
		plugins:     make(map[string]*PluginInfo),
		initialized: newInitializedPlugins(),
	}
	// Create the timer service and set the manager reference
	m.timerService = NewTimerService(m)
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

// getHostLibrary returns the host library (function definitions) for the given host service
func getHostLibrary[S any](
	ctx context.Context,
	instantiateFn func(context.Context, wazero.Runtime, S) error,
	service S,
) (map[string]wazeroapi.FunctionDefinition, error) {
	r := wazero.NewRuntime(ctx)
	if err := instantiateFn(ctx, r, service); err != nil {
		return nil, err
	}
	m := r.Module("env")
	return m.ExportedFunctionDefinitions(), nil
}

// createCustomRuntime returns a function that creates a new wazero runtime with the given compilation cache
// and instantiates the required host functions
func (m *Manager) createCustomRuntime(cache wazero.CompilationCache) api.WazeroNewRuntime {
	return func(ctx context.Context) (wazero.Runtime, error) {
		runtimeConfig := wazero.NewRuntimeConfig().WithCompilationCache(cache)
		r := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)
		if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
			return nil, err
		}

		// Load each host library
		httpLib, err := getHostLibrary[http.HttpService](ctx, http.Instantiate, &HttpServiceImpl{})
		if err != nil {
			return nil, err
		}
		timerLib, err := getHostLibrary[timer.TimerService](ctx, timer.Instantiate, m.timerService)
		if err != nil {
			return nil, err
		}

		// Merge the libraries
		hostLib := maps.Clone(httpLib)
		maps.Copy(hostLib, timerLib)

		// Create the combined host module
		envBuilder := r.NewHostModuleBuilder("env")
		for name, fd := range hostLib {
			fn, ok := fd.GoFunction().(wazeroapi.GoModuleFunction)
			if !ok {
				return nil, fmt.Errorf("invalid function devinition: %s", fd.DebugName())
			}
			envBuilder.NewFunctionBuilder().
				WithGoModuleFunction(fn, fd.ParamTypes(), fd.ResultTypes()).
				WithParameterNames(fd.ParamNames()...).Export(name)
		}

		// Instantiate the combined host module
		if _, err = envBuilder.Instantiate(ctx); err != nil {
			return nil, err
		}
		return r, nil
	}
}

// registerPlugin adds a plugin to the registry with the given parameters
// Used internally by ScanPlugins to register plugins
func (m *Manager) registerPlugin(pluginDir, wasmPath string, manifest *PluginManifest, cache wazero.CompilationCache) *PluginInfo {
	// Create custom runtime function
	customRuntime := m.createCustomRuntime(cache)

	// Configure module and determine plugin name
	mc := newWazeroModuleConfig()

	// Check if it's a symlink, indicating development mode
	isSymlink := false
	if fileInfo, err := os.Lstat(pluginDir); err == nil {
		isSymlink = fileInfo.Mode()&os.ModeSymlink != 0
	}

	// Store plugin info
	state := &pluginState{ready: make(chan struct{})}
	pluginInfo := &PluginInfo{
		Name:         manifest.Name,
		Path:         pluginDir,
		Capabilities: manifest.Capabilities,
		WasmPath:     wasmPath,
		Manifest:     manifest,
		State:        state,
		Runtime:      customRuntime,
		ModConfig:    mc,
	}

	// Start pre-compilation of WASM module in background
	go func() {
		precompilePlugin(state, customRuntime, wasmPath, manifest.Name)

		// Check if this plugin implements InitService and hasn't been initialized yet
		m.initializePluginIfNeeded(pluginInfo)
	}()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.plugins[manifest.Name] = pluginInfo

	log.Info("Discovered plugin", "name", manifest.Name, "capabilities", manifest.Capabilities, "wasm", wasmPath, "dev_mode", isSymlink)
	return m.plugins[manifest.Name]
}

// initializePluginIfNeeded calls OnInit on plugins that implement LifecycleManagement
func (m *Manager) initializePluginIfNeeded(plugin *PluginInfo) {
	// Skip if already initialized
	if m.initialized.isInitialized(plugin) {
		return
	}

	// Check if the plugin implements LifecycleManagement
	for _, capability := range plugin.Capabilities {
		if capability == "LifecycleManagement" {
			m.initialized.callOnInit(plugin)
			m.initialized.markInitialized(plugin)
			break
		}
	}
}

// ScanPlugins scans the plugins directory and compiles all valid plugins without registering them.
func (m *Manager) ScanPlugins() {
	// Get plugins directory from config and read its contents
	root := conf.Server.Plugins.Folder
	log.Debug("Scanning plugins folder", "root", root)
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
	log.Debug("Found entries", "count", len(entries))
	for _, entry := range entries {
		name := entry.Name()
		pluginDir := filepath.Join(root, name)

		// First check if it's a hidden file (starting with .)
		if name[0] == '.' {
			log.Debug("Skipping hidden entry", "name", name)
			continue
		}

		// Check if it's a symlink
		info, err := os.Lstat(pluginDir)
		if err != nil {
			log.Error("Failed to stat entry", "path", pluginDir, err)
			continue
		}

		isSymlink := info.Mode()&os.ModeSymlink != 0
		isDir := info.IsDir()

		log.Debug("Processing entry", "name", name, "isDir", isDir, "isSymlink", isSymlink)

		// Skip if not a directory or symlink
		if !isDir && !isSymlink {
			log.Debug("Skipping non-directory, non-symlink entry", "name", name)
			continue
		}

		// Check if it's a symlink and resolve it if needed
		if isSymlink {
			// Resolve the symlink target
			targetDir, err := os.Readlink(pluginDir)
			if err != nil {
				log.Error("Failed to resolve symlink", "path", pluginDir, err)
				continue
			}
			log.Debug("Processing symlinked plugin directory", "name", name, "path", pluginDir, "target", targetDir)

			// If target is a relative path, make it absolute
			if !filepath.IsAbs(targetDir) {
				targetDir = filepath.Join(filepath.Dir(pluginDir), targetDir)
			}

			// Update the plugin directory to the resolved target
			pluginDir = targetDir
			log.Debug("Updated plugin directory to resolved target", "name", name, "path", pluginDir)

			// Verify that the target is a directory
			targetInfo, err := os.Stat(pluginDir)
			if err != nil {
				log.Error("Failed to stat symlink target", "path", pluginDir, err)
				continue
			}

			if !targetInfo.IsDir() {
				log.Debug("Symlink target is not a directory, skipping", "name", name, "target", pluginDir)
				continue
			}
		}

		wasmPath := filepath.Join(pluginDir, "plugin.wasm")
		log.Debug("Checking for plugin.wasm", "wasmPath", wasmPath)

		// Skip if no WASM file found
		if _, err := os.Stat(wasmPath); err != nil {
			log.Debug("No plugin.wasm found in plugin directory", "plugin", name, "path", wasmPath, "error", err)
			continue
		}

		// Load and validate plugin manifest
		manifestPath := filepath.Join(pluginDir, "manifest.json")
		log.Debug("Loading manifest", "manifestPath", manifestPath)
		manifest, err := LoadManifest(pluginDir)
		if err != nil {
			log.Error("Failed to load manifest", "path", manifestPath, err)
			continue
		}

		if len(manifest.Capabilities) == 0 {
			log.Warn("No capabilities found in plugin manifest", "plugin", name, "path", pluginDir)
			continue
		}
		log.Debug("Manifest loaded successfully", "name", manifest.Name, "capabilities", manifest.Capabilities)

		// Register the plugin
		m.registerPlugin(pluginDir, wasmPath, manifest, cache)
	}
}

// PluginNames returns the names of all plugins that implement the specified capability
func (m *Manager) PluginNames(capability string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var names []string
	for name, plugin := range m.plugins {
		for _, c := range plugin.Capabilities {
			if c == capability {
				names = append(names, name)
				break
			}
		}
	}
	return names
}

func (m *Manager) GetPluginInfo(name string) *PluginInfo {
	m.mu.RLock()
	plugin, ok := m.plugins[name]
	m.mu.RUnlock()

	if !ok {
		log.Warn("Plugin not found", "name", name)
		return nil
	}
	return plugin
}

// LoadPlugin instantiates and returns a plugin by name
func (m *Manager) LoadPlugin(name string, capability string) WasmPlugin {
	plugin := m.GetPluginInfo(name)
	if plugin == nil {
		log.Warn("Plugin not found", "name", name)
		return nil
	}

	log.Debug("Loading plugin", "name", name, "path", plugin.WasmPath)

	if !waitForPluginReady(plugin.State, plugin.Name, plugin.WasmPath) {
		log.Warn("Plugin not ready", "name", name)
		return nil
	}

	if len(plugin.Capabilities) == 0 {
		log.Warn("Plugin has no capabilities", "name", name)
		return nil
	}

	if capability == "" {
		capability = plugin.Capabilities[0]
		log.Debug("No capability specified. Loading first capability", "name", name, "capability", capability)
	}

	if slices.Contains(plugin.Capabilities, capability) {
		log.Trace("Plugin implements capability", "name", name, "capability", capability)
	} else {
		log.Error("Plugin does not implement capability", "name", name, "capability", capability)
		return nil
	}

	// Get the creator function for the capability type
	creator, ok := pluginCreators[capability]
	if !ok {
		log.Warn("Unknown plugin capability type", "capability", capability, "plugin", name)
		return nil
	}

	// Use the creator based on the capability
	adapter := creator(plugin.WasmPath, plugin.Name, plugin.Runtime, plugin.ModConfig)
	if adapter == nil {
		log.Warn("Failed to create adapter for plugin", "name", name)
		return nil
	}
	return adapter
}

// LoadMediaAgent instantiates and returns a media agent plugin by name
func (m *Manager) LoadMediaAgent(name string) (agents.Interface, bool) {
	plugin := m.LoadPlugin(name, CapabilityMetadataAgent)
	if plugin == nil {
		return nil, false
	}
	agent, ok := plugin.(*wasmMediaAgent)
	return agent, ok
}

// LoadScrobbler instantiates and returns a scrobbler plugin by name
func (m *Manager) LoadScrobbler(name string) (scrobbler.Scrobbler, bool) {
	plugin := m.LoadPlugin(name, CapabilityScrobbler)
	if plugin == nil {
		return nil, false
	}
	s, ok := plugin.(scrobbler.Scrobbler)
	return s, ok
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

// LoadAllMediaAgents instantiates and returns all media agent plugins
func (m *Manager) LoadAllMediaAgents() []agents.Interface {
	names := m.PluginNames("MetadataAgent")
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
	names := m.PluginNames("Scrobbler")
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

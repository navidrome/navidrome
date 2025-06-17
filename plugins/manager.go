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
	"maps"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/artwork"
	"github.com/navidrome/navidrome/plugins/host/cache"
	"github.com/navidrome/navidrome/plugins/host/config"
	"github.com/navidrome/navidrome/plugins/host/http"
	"github.com/navidrome/navidrome/plugins/host/scheduler"
	"github.com/navidrome/navidrome/plugins/host/websocket"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/tetratelabs/wazero"
	wazeroapi "github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const (
	CapabilityMetadataAgent       = "MetadataAgent"
	CapabilityScrobbler           = "Scrobbler"
	CapabilitySchedulerCallback   = "SchedulerCallback"
	CapabilityWebSocketCallback   = "WebSocketCallback"
	CapabilityLifecycleManagement = "LifecycleManagement"
)

// pluginCreators maps capability types to their respective creator functions
type pluginConstructor func(wasmPath, pluginName string, runtime api.WazeroNewRuntime, mc wazero.ModuleConfig) WasmPlugin

var pluginCreators = map[string]pluginConstructor{
	CapabilityMetadataAgent:     NewWasmMediaAgent,
	CapabilityScrobbler:         NewWasmScrobblerPlugin,
	CapabilitySchedulerCallback: NewWasmSchedulerCallback,
	CapabilityWebSocketCallback: NewWasmWebSocketCallback,
}

// WasmPlugin is the base interface that all WASM plugins implement
type WasmPlugin interface {
	// PluginName returns the name of the plugin
	PluginName() string
	ServiceType() string
	Instantiate(ctx context.Context) (any, func(), error)
}

const maxParallelCompilations = 2 // Limit to 2 concurrent compilations

var (
	compileSemaphore = make(chan struct{}, maxParallelCompilations)
	compilationCache wazero.CompilationCache
	cacheOnce        sync.Once
	runtimePool      sync.Map // map[string]*pooledRuntime
)

func getCompilationCache() (wazero.CompilationCache, error) {
	var err error
	cacheOnce.Do(func() {
		cacheDir := filepath.Join(conf.Server.CacheFolder, "plugins")
		purgeCacheBySize(cacheDir, conf.Server.Plugins.CacheSize)
		compilationCache, err = wazero.NewCompilationCacheWithDir(cacheDir)
	})
	return compilationCache, err
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
	plugins          map[string]*PluginInfo // Map of plugin name to plugin info
	mu               sync.RWMutex           // Protects plugins map
	schedulerService *schedulerService      // Service for handling scheduled tasks
	websocketService *websocketService      // Service for handling WebSocket connections
	initialized      *initializedPlugins    // Tracks which plugins have been initialized
	adapters         map[string]WasmPlugin  // Map of plugin name + capability to adapter
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

	// Create the host services
	m.schedulerService = newSchedulerService(m)
	m.websocketService = newWebsocketService(m)

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

// loadHostLibrary loads the given host library and returns its exported functions
func loadHostLibrary[S any](
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

// combineLibraries combines the given host libraries into a single "env" module
func (m *Manager) combineLibraries(ctx context.Context, r wazero.Runtime, libs ...map[string]wazeroapi.FunctionDefinition) error {
	// Merge the libraries
	hostLib := map[string]wazeroapi.FunctionDefinition{}
	for _, lib := range libs {
		maps.Copy(hostLib, lib)
	}

	// Create the combined host module
	envBuilder := r.NewHostModuleBuilder("env")
	for name, fd := range hostLib {
		fn, ok := fd.GoFunction().(wazeroapi.GoModuleFunction)
		if !ok {
			return fmt.Errorf("invalid function definition: %s", fd.DebugName())
		}
		envBuilder.NewFunctionBuilder().
			WithGoModuleFunction(fn, fd.ParamTypes(), fd.ResultTypes()).
			WithParameterNames(fd.ParamNames()...).Export(name)
	}

	// Instantiate the combined host module
	if _, err := envBuilder.Instantiate(ctx); err != nil {
		return err
	}
	return nil
}

// createCustomRuntime returns a function that creates a new wazero runtime with the given compilation cache
// and instantiates the required host functions
func (m *Manager) createCustomRuntime(compCache wazero.CompilationCache, pluginName string, permissions map[string]interface{}) api.WazeroNewRuntime {
	return func(ctx context.Context) (wazero.Runtime, error) {
		// Check if runtime already exists
		if rt, ok := runtimePool.Load(pluginName); ok {
			log.Trace(ctx, "Using existing runtime", "plugin", pluginName, "runtime", fmt.Sprintf("%p", rt))
			return rt.(wazero.Runtime), nil
		}

		// Create the runtime
		runtimeConfig := wazero.NewRuntimeConfig().WithCompilationCache(compCache)
		r := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)
		if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
			return nil, err
		}

		// Define all available host services
		type hostService struct {
			name     string
			loadFunc func() (map[string]wazeroapi.FunctionDefinition, error)
		}

		availableServices := []hostService{
			{"config", func() (map[string]wazeroapi.FunctionDefinition, error) {
				return loadHostLibrary[config.ConfigService](ctx, config.Instantiate, &configServiceImpl{pluginName: pluginName})
			}},
			{"http", func() (map[string]wazeroapi.FunctionDefinition, error) {
				return loadHostLibrary[http.HttpService](ctx, http.Instantiate, &httpServiceImpl{pluginName: pluginName})
			}},
			{"scheduler", func() (map[string]wazeroapi.FunctionDefinition, error) {
				return loadHostLibrary[scheduler.SchedulerService](ctx, scheduler.Instantiate, m.schedulerService.HostFunctions(pluginName))
			}},
			{"websocket", func() (map[string]wazeroapi.FunctionDefinition, error) {
				return loadHostLibrary[websocket.WebSocketService](ctx, websocket.Instantiate, m.websocketService.HostFunctions(pluginName))
			}},
			{"cache", func() (map[string]wazeroapi.FunctionDefinition, error) {
				return loadHostLibrary[cache.CacheService](ctx, cache.Instantiate, newCacheService(pluginName))
			}},
			{"artwork", func() (map[string]wazeroapi.FunctionDefinition, error) {
				return loadHostLibrary[artwork.ArtworkService](ctx, artwork.Instantiate, &artworkServiceImpl{})
			}},
		}

		// Load only permitted services
		var grantedPermissions []string
		var libraries []map[string]wazeroapi.FunctionDefinition
		for _, service := range availableServices {
			if _, hasPermission := permissions[service.name]; hasPermission {
				lib, err := service.loadFunc()
				if err != nil {
					return nil, fmt.Errorf("error loading %s lib: %w", service.name, err)
				}
				libraries = append(libraries, lib)
				grantedPermissions = append(grantedPermissions, service.name)
			}
		}
		log.Trace(ctx, "Granting permissions for plugin", "plugin", pluginName, "permissions", grantedPermissions)

		// Combine the permitted libraries
		if err := m.combineLibraries(ctx, r, libraries...); err != nil {
			return nil, err
		}

		pooled := newPooledRuntime(r, pluginName)

		// Use LoadOrStore to atomically check and store, preventing race conditions
		if existing, loaded := runtimePool.LoadOrStore(pluginName, pooled); loaded {
			// Another goroutine created the runtime first, close ours and return the existing one
			log.Trace(ctx, "Race condition detected, using existing runtime", "plugin", pluginName, "runtime", fmt.Sprintf("%p", existing))
			_ = r.Close(ctx)
			return existing.(wazero.Runtime), nil
		}
		log.Trace(ctx, "Created new runtime", "plugin", pluginName, "runtime", fmt.Sprintf("%p", pooled))

		return pooled, nil
	}
}

// registerPlugin adds a plugin to the registry with the given parameters
// Used internally by ScanPlugins to register plugins
func (m *Manager) registerPlugin(pluginDir, wasmPath string, manifest *PluginManifest, cache wazero.CompilationCache) *PluginInfo {
	// Create custom runtime function
	customRuntime := m.createCustomRuntime(cache, manifest.Name, manifest.Permissions)

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

	// Register the plugin
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plugins[manifest.Name] = pluginInfo

	// Register one plugin adapter for each capability
	for _, capability := range manifest.Capabilities {
		constructor := pluginCreators[capability]
		if constructor == nil {
			// Warn about unknown capabilities, except for LifecycleManagement (it does not have an adapter)
			if capability != CapabilityLifecycleManagement {
				log.Warn("Unknown plugin capability type", "capability", capability, "plugin", manifest.Name)
			}
			continue
		}
		adapter := constructor(wasmPath, manifest.Name, customRuntime, mc)
		m.adapters[manifest.Name+"_"+capability] = adapter
	}

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
		if capability == CapabilityLifecycleManagement {
			m.initialized.callOnInit(plugin)
			m.initialized.markInitialized(plugin)
			break
		}
	}
}

// ScanPlugins scans the plugins directory and compiles all valid plugins without registering them.
func (m *Manager) ScanPlugins() {
	// Clear existing plugins
	m.mu.Lock()
	m.plugins = make(map[string]*PluginInfo)
	m.adapters = make(map[string]WasmPlugin)
	m.mu.Unlock()

	// Get plugins directory from config and read its contents
	root := conf.Server.Plugins.Folder
	log.Debug("Scanning plugins folder", "root", root)
	entries, err := os.ReadDir(root)
	if err != nil {
		log.Error("Failed to read plugins folder", "folder", root, err)
		return
	}
	// Get compilation cache to speed up WASM module loading
	ccache, err := getCompilationCache()
	if err != nil {
		log.Error("Failed to initialize plugins compilation cache. Disabling plugins", err)
		return
	}

	// Process each directory in the plugins folder
	log.Debug("Found entries in plugin directory", "count", len(entries), "entries", slice.Map(entries, func(e os.DirEntry) string { return e.Name() }))
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
		m.registerPlugin(pluginDir, wasmPath, manifest, ccache)
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

func (m *Manager) getPlugin(name string, capability string) (*PluginInfo, WasmPlugin) {
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

// LoadPlugin instantiates and returns a plugin by name
func (m *Manager) LoadPlugin(name string, capability string) WasmPlugin {
	info, adapter := m.getPlugin(name, capability)
	if info == nil {
		log.Warn("Plugin not found", "name", name, "capability", capability)
		return nil
	}

	log.Debug("Loading plugin", "name", name, "path", info.WasmPath)

	if !waitForPluginReady(info.State, info.Name, info.WasmPath) {
		log.Warn("Plugin not ready", "name", name, "capability", capability)
		return nil
	}

	if adapter == nil {
		log.Warn("Plugin adapter not found", "name", name, "capability", capability)
		return nil
	}
	return adapter
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

// LoadMediaAgent instantiates and returns a media agent plugin by name
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

// LoadScrobbler instantiates and returns a scrobbler plugin by name
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

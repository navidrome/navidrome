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
	"github.com/navidrome/navidrome/plugins/schema"
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

// PluginDiscoveryEntry represents the result of plugin discovery
type PluginDiscoveryEntry struct {
	ID        string                 // Plugin ID (directory name)
	Path      string                 // Resolved plugin directory path
	WasmPath  string                 // Path to the WASM file
	Manifest  *schema.PluginManifest // Loaded manifest (nil if failed)
	IsSymlink bool                   // Whether the plugin is a development symlink
	Error     error                  // Error encountered during discovery
}

type pluginInfo struct {
	ID           string
	Path         string
	Capabilities []string
	WasmPath     string
	Manifest     *schema.PluginManifest // Loaded manifest
	State        *pluginState
	Runtime      api.WazeroNewRuntime
	ModConfig    wazero.ModuleConfig
}

// Manager is a singleton that manages plugins
type Manager struct {
	plugins          map[string]*pluginInfo  // Map of plugin folder name to plugin info
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
		plugins:   make(map[string]*pluginInfo),
		lifecycle: newPluginLifecycleManager(),
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
func waitForPluginReady(state *pluginState, pluginID, wasmPath string) bool {
	timeout := pluginCompilationTimeout()
	select {
	case <-state.ready:
	case <-time.After(timeout):
		log.Error("Timed out waiting for plugin compilation", "name", pluginID, "path", wasmPath, "timeout", timeout)
		return false
	}
	if state.err != nil {
		log.Error("Failed to compile plugin", "name", pluginID, "path", wasmPath, state.err)
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

func parseTypedPermission[T any](permissions schema.PluginManifestPermissions, permissionName, pluginID string, getter func(schema.PluginManifestPermissions) *T, parser func(*T) (T, error)) (T, error) {
	var parsed T
	if permData := getter(permissions); permData != nil {
		var err error
		parsed, err = parser(permData)
		if err != nil {
			return parsed, fmt.Errorf("invalid %s permissions for plugin %s: %w", permissionName, pluginID, err)
		}
	}
	return parsed, nil
}

// createCustomRuntime returns a function that creates a new wazero runtime with the given compilation cache
// and instantiates the required host functions
func (m *Manager) createCustomRuntime(compCache wazero.CompilationCache, pluginID string, permissions schema.PluginManifestPermissions) api.WazeroNewRuntime {
	return func(ctx context.Context) (wazero.Runtime, error) {
		// Check if runtime already exists
		if rt, ok := runtimePool.Load(pluginID); ok {
			log.Trace(ctx, "Using existing runtime", "plugin", pluginID, "runtime", fmt.Sprintf("%p", rt))
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
				return loadHostLibrary[config.ConfigService](ctx, config.Instantiate, &configServiceImpl{pluginID: pluginID})
			}},
			{"http", func() (map[string]wazeroapi.FunctionDefinition, error) {
				if permissions.Http == nil {
					return nil, fmt.Errorf("http permissions not granted")
				}
				httpPerms, err := parseHTTPPermissions(permissions.Http)
				if err != nil {
					return nil, fmt.Errorf("invalid http permissions for plugin %s: %w", pluginID, err)
				}
				return loadHostLibrary[http.HttpService](ctx, http.Instantiate, &httpServiceImpl{
					pluginID:    pluginID,
					permissions: httpPerms,
				})
			}},
			{"scheduler", func() (map[string]wazeroapi.FunctionDefinition, error) {
				return loadHostLibrary[scheduler.SchedulerService](ctx, scheduler.Instantiate, m.schedulerService.HostFunctions(pluginID))
			}},
			{"websocket", func() (map[string]wazeroapi.FunctionDefinition, error) {
				if permissions.Websocket == nil {
					return nil, fmt.Errorf("websocket permissions not granted")
				}
				wsPerms, err := parseWebSocketPermissions(permissions.Websocket)
				if err != nil {
					return nil, fmt.Errorf("invalid websocket permissions for plugin %s: %w", pluginID, err)
				}
				return loadHostLibrary[websocket.WebSocketService](ctx, websocket.Instantiate, m.websocketService.HostFunctions(pluginID, wsPerms))
			}},
			{"cache", func() (map[string]wazeroapi.FunctionDefinition, error) {
				return loadHostLibrary[cache.CacheService](ctx, cache.Instantiate, newCacheService(pluginID))
			}},
			{"artwork", func() (map[string]wazeroapi.FunctionDefinition, error) {
				return loadHostLibrary[artwork.ArtworkService](ctx, artwork.Instantiate, &artworkServiceImpl{})
			}},
		}

		// Load only permitted services
		var grantedPermissions []string
		var libraries []map[string]wazeroapi.FunctionDefinition
		for _, service := range availableServices {
			var hasPermission bool
			switch service.name {
			case "config":
				hasPermission = permissions.Config != nil
			case "http":
				hasPermission = permissions.Http != nil
			case "scheduler":
				hasPermission = permissions.Scheduler != nil
			case "websocket":
				hasPermission = permissions.Websocket != nil
			case "cache":
				hasPermission = permissions.Cache != nil
			case "artwork":
				hasPermission = permissions.Artwork != nil
			}

			if hasPermission {
				lib, err := service.loadFunc()
				if err != nil {
					return nil, fmt.Errorf("error loading %s lib: %w", service.name, err)
				}
				libraries = append(libraries, lib)
				grantedPermissions = append(grantedPermissions, service.name)
			}
		}
		log.Trace(ctx, "Granting permissions for plugin", "plugin", pluginID, "permissions", grantedPermissions)

		// Combine the permitted libraries
		if err := m.combineLibraries(ctx, r, libraries...); err != nil {
			return nil, err
		}

		pooled := newPooledRuntime(r, pluginID)

		// Use LoadOrStore to atomically check and store, preventing race conditions
		if existing, loaded := runtimePool.LoadOrStore(pluginID, pooled); loaded {
			// Another goroutine created the runtime first, close ours and return the existing one
			log.Trace(ctx, "Race condition detected, using existing runtime", "plugin", pluginID, "runtime", fmt.Sprintf("%p", existing))
			_ = r.Close(ctx)
			return existing.(wazero.Runtime), nil
		}
		log.Trace(ctx, "Created new runtime", "plugin", pluginID, "runtime", fmt.Sprintf("%p", pooled))

		return pooled, nil
	}
}

// registerPlugin adds a plugin to the registry with the given parameters
// Used internally by ScanPlugins to register plugins
func (m *Manager) registerPlugin(pluginID, pluginDir, wasmPath string, manifest *schema.PluginManifest, cache wazero.CompilationCache) *pluginInfo {
	// Create custom runtime function
	customRuntime := m.createCustomRuntime(cache, pluginID, manifest.Permissions)

	// Configure module and determine plugin name
	mc := newWazeroModuleConfig()

	// Check if it's a symlink, indicating development mode
	isSymlink := false
	if fileInfo, err := os.Lstat(pluginDir); err == nil {
		isSymlink = fileInfo.Mode()&os.ModeSymlink != 0
	}

	// Store plugin info
	state := &pluginState{ready: make(chan struct{})}
	pluginInfo := &pluginInfo{
		ID:           pluginID,
		Path:         pluginDir,
		Capabilities: slice.Map(manifest.Capabilities, func(cap schema.PluginManifestCapabilitiesElem) string { return string(cap) }),
		WasmPath:     wasmPath,
		Manifest:     manifest,
		State:        state,
		Runtime:      customRuntime,
		ModConfig:    mc,
	}

	// Start pre-compilation of WASM module in background
	go func() {
		precompilePlugin(state, customRuntime, wasmPath, pluginID)

		// Check if this plugin implements InitService and hasn't been initialized yet
		m.initializePluginIfNeeded(pluginInfo)
	}()

	// Register the plugin
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plugins[pluginID] = pluginInfo

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
func (m *Manager) initializePluginIfNeeded(plugin *pluginInfo) {
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

// DiscoverPlugins scans the plugins directory and returns information about all discoverable plugins
// This shared function eliminates duplication between ScanPlugins and plugin list commands
func DiscoverPlugins(pluginsDir string) []PluginDiscoveryEntry {
	var discoveries []PluginDiscoveryEntry

	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		// Return a single entry with the error
		return []PluginDiscoveryEntry{{
			Error: fmt.Errorf("failed to read plugins directory %s: %w", pluginsDir, err),
		}}
	}

	for _, entry := range entries {
		name := entry.Name()
		pluginPath := filepath.Join(pluginsDir, name)

		// Skip hidden files
		if name[0] == '.' {
			continue
		}

		// Check if it's a directory or symlink
		info, err := os.Lstat(pluginPath)
		if err != nil {
			discoveries = append(discoveries, PluginDiscoveryEntry{
				ID:    name,
				Error: fmt.Errorf("failed to stat entry %s: %w", pluginPath, err),
			})
			continue
		}

		isSymlink := info.Mode()&os.ModeSymlink != 0
		isDir := info.IsDir()

		// Skip if not a directory or symlink
		if !isDir && !isSymlink {
			continue
		}

		// Resolve symlinks
		pluginDir := pluginPath
		if isSymlink {
			targetDir, err := os.Readlink(pluginPath)
			if err != nil {
				discoveries = append(discoveries, PluginDiscoveryEntry{
					ID:        name,
					IsSymlink: true,
					Error:     fmt.Errorf("failed to resolve symlink %s: %w", pluginPath, err),
				})
				continue
			}

			// If target is a relative path, make it absolute
			if !filepath.IsAbs(targetDir) {
				targetDir = filepath.Join(filepath.Dir(pluginPath), targetDir)
			}

			// Verify that the target is a directory
			targetInfo, err := os.Stat(targetDir)
			if err != nil {
				discoveries = append(discoveries, PluginDiscoveryEntry{
					ID:        name,
					IsSymlink: true,
					Error:     fmt.Errorf("failed to stat symlink target %s: %w", targetDir, err),
				})
				continue
			}

			if !targetInfo.IsDir() {
				discoveries = append(discoveries, PluginDiscoveryEntry{
					ID:        name,
					IsSymlink: true,
					Error:     fmt.Errorf("symlink target is not a directory: %s", targetDir),
				})
				continue
			}

			pluginDir = targetDir
		}

		// Check for WASM file
		wasmPath := filepath.Join(pluginDir, "plugin.wasm")
		if _, err := os.Stat(wasmPath); err != nil {
			discoveries = append(discoveries, PluginDiscoveryEntry{
				ID:    name,
				Path:  pluginDir,
				Error: fmt.Errorf("no plugin.wasm found: %w", err),
			})
			continue
		}

		// Load manifest
		manifest, err := LoadManifest(pluginDir)
		if err != nil {
			discoveries = append(discoveries, PluginDiscoveryEntry{
				ID:    name,
				Path:  pluginDir,
				Error: fmt.Errorf("failed to load manifest: %w", err),
			})
			continue
		}

		// Check for capabilities
		if len(manifest.Capabilities) == 0 {
			discoveries = append(discoveries, PluginDiscoveryEntry{
				ID:    name,
				Path:  pluginDir,
				Error: fmt.Errorf("no capabilities found in manifest"),
			})
			continue
		}

		// Success!
		discoveries = append(discoveries, PluginDiscoveryEntry{
			ID:        name,
			Path:      pluginDir,
			WasmPath:  wasmPath,
			Manifest:  manifest,
			IsSymlink: isSymlink,
		})
	}

	return discoveries
}

// ScanPlugins scans the plugins directory and compiles all valid plugins without registering them.
func (m *Manager) ScanPlugins() {
	// Clear existing plugins
	m.mu.Lock()
	m.plugins = make(map[string]*pluginInfo)
	m.adapters = make(map[string]WasmPlugin)
	m.mu.Unlock()

	// Get plugins directory from config
	root := conf.Server.Plugins.Folder
	log.Debug("Scanning plugins folder", "root", root)

	// Get compilation cache to speed up WASM module loading
	ccache, err := getCompilationCache()
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
		m.registerPlugin(discovery.ID, discovery.Path, discovery.WasmPath, discovery.Manifest, ccache)
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

func (m *Manager) getPlugin(name string, capability string) (*pluginInfo, WasmPlugin) {
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

	if !waitForPluginReady(info.State, info.ID, info.WasmPath) {
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

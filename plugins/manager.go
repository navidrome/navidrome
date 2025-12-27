package plugins

import (
	"cmp"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dustin/go-humanize"
	extism "github.com/extism/go-sdk"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/scheduler"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/rjeczalik/notify"
	"github.com/tetratelabs/wazero"
	"golang.org/x/sync/errgroup"
)

const (
	// manifestFunction is the name of the function that plugins must export
	// to provide their manifest.
	manifestFunction = "nd_manifest"

	// defaultTimeout is the default timeout for plugin function calls
	defaultTimeout = 30 * time.Second

	// maxPluginLoadConcurrency is the maximum number of plugins that can be
	// compiled/loaded in parallel during startup
	maxPluginLoadConcurrency = 3
)

// SubsonicRouter is an http.Handler that serves Subsonic API requests.
type SubsonicRouter = http.Handler

// Manager manages loading and lifecycle of WebAssembly plugins.
// It implements both agents.PluginLoader and scrobbler.PluginLoader interfaces.
type Manager struct {
	mu      sync.RWMutex
	plugins map[string]*plugin
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

	// SubsonicAPI host function dependencies (set once before Start, not modified after)
	subsonicRouter SubsonicRouter
	ds             model.DataStore
}

// plugin represents a loaded plugin
type plugin struct {
	name         string // Plugin name (from filename)
	path         string // Path to the wasm file
	manifest     *Manifest
	compiled     *extism.CompiledPlugin
	capabilities []Capability // Auto-detected capabilities based on exported functions
	closers      []io.Closer  // Cleanup functions to call on unload
}

func (p *plugin) instance() (*extism.Plugin, error) {
	instance, err := p.compiled.Instance(context.Background(), extism.PluginInstanceConfig{
		ModuleConfig: wazero.NewModuleConfig().WithSysWalltime().WithRandSource(rand.Reader),
	})
	if err != nil {
		return nil, err
	}
	instance.SetLogger(extismLogger(p.name))
	return instance, nil
}

func (p *plugin) Close() error {
	var errs []error
	for _, f := range p.closers {
		err := f.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// GetManager returns a singleton instance of the plugin manager.
// The manager is not started automatically; call Start() to begin loading plugins.
func GetManager() *Manager {
	return singleton.GetInstance(func() *Manager {
		return &Manager{
			plugins: make(map[string]*plugin),
		}
	})
}

// SetSubsonicRouter sets the Subsonic router for SubsonicAPI host functions.
// This should be called after the subsonic router is created but before plugins
// that require SubsonicAPI access are loaded.
func (m *Manager) SetSubsonicRouter(router SubsonicRouter) {
	m.subsonicRouter = router
}

// SetDataStore sets the data store for plugins that need database access.
// This should be called before plugins are loaded.
func (m *Manager) SetDataStore(ds model.DataStore) {
	m.ds = ds
}

// Start initializes the plugin manager and loads plugins from the configured folder.
// It should be called once during application startup when plugins are enabled.
func (m *Manager) Start(ctx context.Context) error {
	if !conf.Server.Plugins.Enabled {
		log.Debug(ctx, "Plugin system is disabled")
		return nil
	}

	// Set extism log level based on plugin-specific config or global log level
	pluginLogLevel := conf.Server.Plugins.LogLevel
	if pluginLogLevel == "" {
		pluginLogLevel = conf.Server.LogLevel
	}
	extism.SetLogLevel(toExtismLogLevel(log.ParseLogLevel(pluginLogLevel)))

	m.ctx, m.cancel = context.WithCancel(ctx)

	// Initialize wazero compilation cache for better performance
	cacheDir := filepath.Join(conf.Server.CacheFolder, "plugins")
	purgeCacheBySize(ctx, cacheDir, conf.Server.Plugins.CacheSize)

	var err error
	m.cache, err = wazero.NewCompilationCacheWithDir(cacheDir)
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
	for name, plugin := range m.plugins {
		err := plugin.Close()
		if err != nil {
			log.Error("Error during plugin cleanup", "plugin", name, err)
		}
		if plugin.compiled != nil {
			if err := plugin.compiled.Close(context.Background()); err != nil {
				log.Error("Error closing plugin", "plugin", name, err)
			}
		}
	}
	m.plugins = make(map[string]*plugin)

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
	for name, plugin := range m.plugins {
		if hasCapability(plugin.capabilities, cap) {
			names = append(names, name)
		}
	}
	return names
}

// LoadMediaAgent loads and returns a media agent plugin by name.
// Returns false if the plugin is not found or doesn't have the MetadataAgent capability.
func (m *Manager) LoadMediaAgent(name string) (agents.Interface, bool) {
	m.mu.RLock()
	plugin, ok := m.plugins[name]
	m.mu.RUnlock()

	if !ok || !hasCapability(plugin.capabilities, CapabilityMetadataAgent) {
		return nil, false
	}

	// Create a new metadata agent adapter for this plugin
	return &MetadataAgent{
		name:   plugin.name,
		plugin: plugin,
	}, true
}

// LoadScrobbler loads and returns a scrobbler plugin by name.
// Returns false if the plugin is not found or doesn't have the Scrobbler capability.
func (m *Manager) LoadScrobbler(name string) (scrobbler.Scrobbler, bool) {
	m.mu.RLock()
	plugin, ok := m.plugins[name]
	m.mu.RUnlock()

	if !ok || !hasCapability(plugin.capabilities, CapabilityScrobbler) {
		return nil, false
	}

	// Create a new scrobbler adapter for this plugin
	return &ScrobblerPlugin{
		name:   plugin.name,
		plugin: plugin,
	}, true
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
	for name, plugin := range m.plugins {
		info[name] = PluginInfo{
			Name:    plugin.manifest.Name,
			Version: plugin.manifest.Version,
		}
	}
	return info
}

// discoverPlugins scans the plugins folder and loads all .wasm files in parallel
func (m *Manager) discoverPlugins(folder string) error {
	entries, err := os.ReadDir(folder)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug("Plugins folder does not exist", "folder", folder)
			return nil
		}
		return err
	}

	// Collect all plugin files to load
	type pluginFile struct {
		name string
		path string
	}
	var pluginFiles []pluginFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".wasm") {
			continue
		}
		pluginFiles = append(pluginFiles, pluginFile{
			name: strings.TrimSuffix(entry.Name(), ".wasm"),
			path: filepath.Join(folder, entry.Name()),
		})
	}

	if len(pluginFiles) == 0 {
		log.Trace(m.ctx, "No plugins found", "folder", folder)
		return nil
	}

	g := errgroup.Group{}
	g.SetLimit(maxPluginLoadConcurrency)

	for _, pf := range pluginFiles {
		g.Go(func() error {
			start := time.Now()
			log.Debug(m.ctx, "Loading plugin", "plugin", pf.name, "path", pf.path)
			defer func() {
				log.Debug(m.ctx, "Finished loading plugin", "plugin", pf.name, "duration", time.Since(start))
			}()

			// Panic recovery to prevent one plugin from crashing the loading process
			defer func() {
				if r := recover(); r != nil {
					log.Error(m.ctx, "Panic while loading plugin", "plugin", pf.name, "panic", r)
				}
			}()

			if err := m.loadPlugin(pf.name, pf.path); err != nil {
				log.Error(m.ctx, "Failed to load plugin", "plugin", pf.name, "path", pf.path, err)
				return nil
			}

			m.mu.RLock()
			p := m.plugins[pf.name]
			m.mu.RUnlock()
			if p != nil {
				log.Info(m.ctx, "Loaded plugin", "plugin", pf.name, "manifest", p.manifest.Name, "capabilities", p.capabilities)
			}
			return nil
		})
	}

	return g.Wait()
}

// loadPlugin loads a single plugin from a wasm file
func (m *Manager) loadPlugin(name, wasmPath string) error {
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

	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return err
	}

	pluginConfig := m.getPluginConfig(name)
	pluginManifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmData{Data: wasmBytes, Name: "main"},
		},
		Config:  pluginConfig,
		Timeout: uint64(defaultTimeout.Milliseconds()),
	}
	extismConfig := extism.PluginConfig{
		EnableWasi:    true,
		RuntimeConfig: wazero.NewRuntimeConfig().WithCompilationCache(m.cache),
	}

	// Register stub host functions for initial compilation.
	// This is necessary because plugins that import host functions will fail to compile if those
	// functions aren't available at compile time.
	// The real service will be registered during recompilation.
	stubHostFunctions := host.RegisterSubsonicAPIHostFunctions(nil)
	stubHostFunctions = append(stubHostFunctions, host.RegisterSchedulerHostFunctions(nil)...)
	stubHostFunctions = append(stubHostFunctions, host.RegisterWebSocketHostFunctions(nil)...)
	stubHostFunctions = append(stubHostFunctions, host.RegisterArtworkHostFunctions(nil)...)
	stubHostFunctions = append(stubHostFunctions, host.RegisterCacheHostFunctions(nil)...)

	// Create initial compiled plugin with stub host functions
	compiled, err := extism.NewCompiledPlugin(m.ctx, pluginManifest, extismConfig, stubHostFunctions)
	if err != nil {
		return err
	}

	// Create instance to read manifest and detect capabilities
	instance, err := compiled.Instance(m.ctx, extism.PluginInstanceConfig{})
	if err != nil {
		compiled.Close(m.ctx)
		return err
	}
	instance.SetLogger(extismLogger(name))

	exit, manifestBytes, err := instance.Call(manifestFunction, nil)
	if err != nil {
		instance.Close(m.ctx)
		compiled.Close(m.ctx)
		return err
	}
	if exit != 0 {
		instance.Close(m.ctx)
		compiled.Close(m.ctx)
		return fmt.Errorf("calling %s: %d", manifestFunction, exit)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		instance.Close(m.ctx)
		compiled.Close(m.ctx)
		return fmt.Errorf("invalid plugin manifest: %w", err)
	}

	// Detect capabilities using the instance before closing it
	capabilities := detectCapabilities(instance)
	instance.Close(m.ctx)

	var hostFunctions []extism.HostFunction
	var closers []io.Closer

	if hosts := manifest.AllowedHosts(); len(hosts) > 0 {
		pluginManifest.AllowedHosts = hosts
	}

	// Register SubsonicAPI host functions if permission is granted
	if manifest.Permissions != nil && manifest.Permissions.Subsonicapi != nil {
		perm := manifest.Permissions.Subsonicapi
		if m.subsonicRouter != nil && m.ds != nil {
			service := newSubsonicAPIService(name, m.subsonicRouter, m.ds, perm)
			hostFunctions = append(hostFunctions, host.RegisterSubsonicAPIHostFunctions(service)...)
		} else {
			log.Warn(m.ctx, "Plugin requires SubsonicAPI but router/datastore not available", "plugin", name)
		}
	}

	// Register Scheduler host functions if permission is granted
	if manifest.Permissions != nil && manifest.Permissions.Scheduler != nil {
		service := newSchedulerService(name, m, scheduler.GetInstance())
		closers = append(closers, service)
		hostFunctions = append(hostFunctions, host.RegisterSchedulerHostFunctions(service)...)
	}

	// Register WebSocket host functions if permission is granted
	if manifest.Permissions != nil && manifest.Permissions.Websocket != nil {
		perm := manifest.Permissions.Websocket
		service := newWebSocketService(name, m, perm.AllowedHosts)
		closers = append(closers, service)
		hostFunctions = append(hostFunctions, host.RegisterWebSocketHostFunctions(service)...)
	}

	// Register Artwork host functions if permission is granted
	if manifest.Permissions != nil && manifest.Permissions.Artwork != nil {
		service := newArtworkService()
		hostFunctions = append(hostFunctions, host.RegisterArtworkHostFunctions(service)...)
	}

	// Register Cache host functions if permission is granted
	if manifest.Permissions != nil && manifest.Permissions.Cache != nil {
		service := newCacheService(name)
		closers = append(closers, service)
		hostFunctions = append(hostFunctions, host.RegisterCacheHostFunctions(service)...)
	}

	// Check if the plugin needs to be recompiled with real host functions
	needsRecompile := len(pluginManifest.AllowedHosts) > 0 || len(hostFunctions) > 0

	// Recompile if needed. It is actually not a "recompile" since the first compilation
	// should be cached by wazero. We just need to do it this way to provide the real host functions.
	if needsRecompile {
		log.Trace(m.ctx, "Recompiling plugin", "plugin", name)
		compiled.Close(m.ctx)
		compiled, err = extism.NewCompiledPlugin(m.ctx, pluginManifest, extismConfig, hostFunctions)
		if err != nil {
			return err
		}
	}

	m.mu.Lock()
	m.plugins[name] = &plugin{
		name:         name,
		path:         wasmPath,
		manifest:     &manifest,
		compiled:     compiled,
		capabilities: capabilities,
		closers:      closers,
	}
	m.mu.Unlock()

	// Call plugin init function if the plugin has the Lifecycle capability
	callPluginInit(m.ctx, m.plugins[name])

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
	plugin, ok := m.plugins[name]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("plugin %q not found", name)
	}
	delete(m.plugins, name)
	m.mu.Unlock()

	// Run cleanup functions
	err := plugin.Close()
	if err != nil {
		log.Error("Error during plugin cleanup", "plugin", name, err)
	}

	// Close the compiled plugin outside the lock with a grace period
	// to allow in-flight requests to complete
	if plugin.compiled != nil {
		// Use a brief timeout for cleanup
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := plugin.compiled.Close(ctx); err != nil {
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
	return nil
}

var errFunctionNotFound = errors.New("function not found")

// callPluginFunction is a helper to call a plugin function with input and output types.
// It handles JSON marshalling/unmarshalling and error checking.
func callPluginFunction[I any, O any](ctx context.Context, plugin *plugin, funcName string, input I) (O, error) {
	start := time.Now()

	var result O

	// Create plugin instance
	p, err := plugin.instance()
	if err != nil {
		return result, fmt.Errorf("failed to create plugin: %w", err)
	}
	defer p.Close(ctx)

	if !p.FunctionExists(funcName) {
		log.Trace(ctx, "Plugin function not found", "plugin", plugin.name, "function", funcName)
		return result, fmt.Errorf("%w: %s", errFunctionNotFound, funcName)
	}

	inputBytes, err := json.Marshal(input)
	if err != nil {
		return result, fmt.Errorf("failed to marshal input: %w", err)
	}

	startCall := time.Now()
	exit, output, err := p.Call(funcName, inputBytes)
	if err != nil {
		log.Trace(ctx, "Plugin call failed", "plugin", plugin.name, "function", funcName, "pluginDuration", time.Since(startCall), "navidromeDuration", startCall.Sub(start), err)
		return result, fmt.Errorf("plugin call failed: %w", err)
	}
	if exit != 0 {
		return result, fmt.Errorf("plugin call exited with code %d", exit)
	}

	if len(output) > 0 {
		err = json.Unmarshal(output, &result)
		if err != nil {
			log.Trace(ctx, "Plugin call failed", "plugin", plugin.name, "function", funcName, "pluginDuration", time.Since(startCall), "navidromeDuration", startCall.Sub(start), err)
		}
	}

	log.Trace(ctx, "Plugin call succeeded", "plugin", plugin.name, "function", funcName, "pluginDuration", time.Since(startCall), "navidromeDuration", startCall.Sub(start))
	return result, err
}

// extismLogger is a helper to log messages from Extism plugins
func extismLogger(pluginName string) func(level extism.LogLevel, msg string) {
	return func(level extism.LogLevel, msg string) {
		if level == extism.LogLevelOff {
			return
		}
		log.Log(log.ParseLogLevel(level.String()), msg, "plugin", pluginName)
	}
}

// toExtismLogLevel converts a Navidrome log level to an extism LogLevel
func toExtismLogLevel(level log.Level) extism.LogLevel {
	switch level {
	case log.LevelTrace:
		return extism.LogLevelTrace
	case log.LevelDebug:
		return extism.LogLevelDebug
	case log.LevelInfo:
		return extism.LogLevelInfo
	case log.LevelWarn:
		return extism.LogLevelWarn
	case log.LevelError, log.LevelFatal:
		return extism.LogLevelError
	default:
		return extism.LogLevelInfo
	}
}

// purgeCacheBySize removes the oldest files in dir until its total size is
// lower than or equal to maxSize. maxSize should be a human-readable string
// like "10MB" or "200K". If parsing fails or maxSize is "0", the function is
// a no-op.
func purgeCacheBySize(ctx context.Context, dir, maxSize string) {
	sizeLimit, err := humanize.ParseBytes(maxSize)
	if err != nil || sizeLimit == 0 {
		return
	}

	type fileInfo struct {
		path string
		size uint64
		mod  int64
	}

	var files []fileInfo
	var total uint64

	walk := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Trace(ctx, "Failed to access plugin cache entry", "path", path, err)
			return nil //nolint:nilerr
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			log.Trace(ctx, "Failed to get file info for plugin cache entry", "path", path, err)
			return nil //nolint:nilerr
		}
		files = append(files, fileInfo{
			path: path,
			size: uint64(info.Size()),
			mod:  info.ModTime().UnixMilli(),
		})
		total += uint64(info.Size())
		return nil
	}

	if err := filepath.WalkDir(dir, walk); err != nil {
		if !os.IsNotExist(err) {
			log.Warn(ctx, "Failed to traverse plugin cache directory", "path", dir, err)
		}
		return
	}

	log.Trace(ctx, "Current plugin cache size", "path", dir, "size", humanize.Bytes(total), "sizeLimit", humanize.Bytes(sizeLimit))
	if total <= sizeLimit {
		return
	}

	log.Debug(ctx, "Purging plugin cache", "path", dir, "sizeLimit", humanize.Bytes(sizeLimit), "currentSize", humanize.Bytes(total))
	slices.SortFunc(files, func(i, j fileInfo) int { return cmp.Compare(i.mod, j.mod) })

	for _, f := range files {
		if total <= sizeLimit {
			break
		}
		if err := os.Remove(f.path); err != nil {
			log.Warn(ctx, "Failed to remove plugin cache entry", "path", f.path, "size", humanize.Bytes(f.size), err)
			continue
		}
		total -= f.size
		log.Debug(ctx, "Removed plugin cache entry", "path", f.path, "size", humanize.Bytes(f.size), "time", time.UnixMilli(f.mod), "remainingSize", humanize.Bytes(total))

		// Remove empty parent directories
		dirPath := filepath.Dir(f.path)
		for dirPath != dir {
			if err := os.Remove(dirPath); err != nil {
				break
			}
			dirPath = filepath.Dir(dirPath)
		}
	}
}

package plugins

import (
	"cmp"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
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
	"github.com/navidrome/navidrome/model/request"
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
func GetManager(ds model.DataStore) *Manager {
	return singleton.GetInstance(func() *Manager {
		return &Manager{
			ds:      ds,
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

// Start initializes the plugin manager and loads plugins from the configured folder.
// It should be called once during application startup when plugins are enabled.
// The startup flow is:
// 1. Sync plugins folder with DB (discover new, update changed, remove deleted)
// 2. Load only enabled plugins from DB
func (m *Manager) Start(ctx context.Context) error {
	if !conf.Server.Plugins.Enabled {
		log.Debug(ctx, "Plugin system is disabled")
		return nil
	}

	if m.subsonicRouter == nil {
		log.Fatal(ctx, "Plugin manager requires DataStore to be configured")
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

	// Sync plugins folder with DB
	if err := m.SyncPlugins(ctx, folder); err != nil {
		log.Error(ctx, "Error syncing plugins with DB", err)
		// Continue - we can still try to load plugins
	}

	// Load enabled plugins from DB
	if err := m.loadEnabledPlugins(ctx); err != nil {
		log.Error(ctx, "Error loading enabled plugins", err)
		return fmt.Errorf("loading enabled plugins: %w", err)
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

// adminContext returns a context with admin privileges for DB operations.
func adminContext(ctx context.Context) context.Context {
	return request.WithUser(ctx, model.User{IsAdmin: true})
}

// marshalManifest marshals a manifest to JSON string, returning empty string on error.
func marshalManifest(m *Manifest) string {
	b, _ := json.Marshal(m)
	return string(b)
}

// addPluginToDB adds a new plugin to the database as disabled.
func (m *Manager) addPluginToDB(ctx context.Context, repo model.PluginRepository, name, path string, metadata *PluginMetadata) error {
	now := time.Now()
	newPlugin := &model.Plugin{
		ID:        name,
		Path:      path,
		Manifest:  marshalManifest(metadata.Manifest),
		SHA256:    metadata.SHA256,
		Enabled:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Put(newPlugin); err != nil {
		return fmt.Errorf("adding plugin to DB: %w", err)
	}
	log.Info(ctx, "Discovered new plugin", "plugin", name)
	return nil
}

// updatePluginInDB updates an existing plugin in the database after a file change.
// If the plugin was enabled, it will be unloaded and disabled.
func (m *Manager) updatePluginInDB(ctx context.Context, repo model.PluginRepository, dbPlugin *model.Plugin, path string, metadata *PluginMetadata) error {
	wasEnabled := dbPlugin.Enabled
	if wasEnabled {
		if err := m.UnloadPlugin(dbPlugin.ID); err != nil {
			log.Debug(ctx, "Plugin not loaded during change", "plugin", dbPlugin.ID, err)
		}
	}
	dbPlugin.Path = path
	dbPlugin.Manifest = marshalManifest(metadata.Manifest)
	dbPlugin.SHA256 = metadata.SHA256
	dbPlugin.Enabled = false
	dbPlugin.LastError = ""
	dbPlugin.UpdatedAt = time.Now()
	if err := repo.Put(dbPlugin); err != nil {
		return fmt.Errorf("updating plugin in DB: %w", err)
	}
	log.Info(ctx, "Plugin file changed", "plugin", dbPlugin.ID, "wasEnabled", wasEnabled)
	return nil
}

// removePluginFromDB removes a plugin from the database.
// If the plugin was enabled, it will be unloaded first.
func (m *Manager) removePluginFromDB(ctx context.Context, repo model.PluginRepository, dbPlugin *model.Plugin) error {
	if dbPlugin.Enabled {
		if err := m.UnloadPlugin(dbPlugin.ID); err != nil {
			log.Debug(ctx, "Plugin not loaded during removal", "plugin", dbPlugin.ID, err)
		}
	}
	if err := repo.Delete(dbPlugin.ID); err != nil {
		return fmt.Errorf("deleting plugin from DB: %w", err)
	}
	log.Info(ctx, "Plugin removed", "plugin", dbPlugin.ID)
	return nil
}

// PluginMetadata holds the extracted information from a plugin file
// without fully initializing the plugin.
type PluginMetadata struct {
	Manifest *Manifest
	SHA256   string
}

// computeFileSHA256 computes the SHA-256 hash of a file without loading it into memory.
// This is used for quick change detection before full plugin compilation.
func computeFileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// compiledPluginInfo holds the intermediate compilation result used by both
// ExtractManifest and loadPluginWithConfig.
type compiledPluginInfo struct {
	wasmBytes []byte
	sha256    string
	manifest  *Manifest
	compiled  *extism.CompiledPlugin
}

// serviceContext provides dependencies needed by host service factories.
type serviceContext struct {
	pluginName  string
	manager     *Manager
	permissions *Permissions
}

// hostServiceEntry defines a host service for table-driven registration.
type hostServiceEntry struct {
	name          string
	hasPermission func(*Permissions) bool
	registerStubs func() []extism.HostFunction
	create        func(*serviceContext) ([]extism.HostFunction, io.Closer)
}

// hostServices defines all available host services.
// Adding a new host service only requires adding an entry here.
var hostServices = []hostServiceEntry{
	{
		name:          "SubsonicAPI",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Subsonicapi != nil },
		registerStubs: func() []extism.HostFunction { return host.RegisterSubsonicAPIHostFunctions(nil) },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer) {
			perm := ctx.permissions.Subsonicapi
			service := newSubsonicAPIService(ctx.pluginName, ctx.manager.subsonicRouter, ctx.manager.ds, perm)
			return host.RegisterSubsonicAPIHostFunctions(service), nil
		},
	},
	{
		name:          "Scheduler",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Scheduler != nil },
		registerStubs: func() []extism.HostFunction { return host.RegisterSchedulerHostFunctions(nil) },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer) {
			service := newSchedulerService(ctx.pluginName, ctx.manager, scheduler.GetInstance())
			return host.RegisterSchedulerHostFunctions(service), service
		},
	},
	{
		name:          "WebSocket",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Websocket != nil },
		registerStubs: func() []extism.HostFunction { return host.RegisterWebSocketHostFunctions(nil) },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer) {
			perm := ctx.permissions.Websocket
			service := newWebSocketService(ctx.pluginName, ctx.manager, perm.AllowedHosts)
			return host.RegisterWebSocketHostFunctions(service), service
		},
	},
	{
		name:          "Artwork",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Artwork != nil },
		registerStubs: func() []extism.HostFunction { return host.RegisterArtworkHostFunctions(nil) },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer) {
			service := newArtworkService()
			return host.RegisterArtworkHostFunctions(service), nil
		},
	},
	{
		name:          "Cache",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Cache != nil },
		registerStubs: func() []extism.HostFunction { return host.RegisterCacheHostFunctions(nil) },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer) {
			service := newCacheService(ctx.pluginName)
			return host.RegisterCacheHostFunctions(service), service
		},
	},
}

// stubHostFunctions returns the list of stub host functions needed for initial plugin compilation.
func stubHostFunctions() []extism.HostFunction {
	var stubs []extism.HostFunction
	for _, entry := range hostServices {
		stubs = append(stubs, entry.registerStubs()...)
	}
	return stubs
}

// compileAndExtractManifest reads a wasm file, compiles it with cache, and extracts the manifest.
// The caller is responsible for closing the returned compiled plugin when done.
func (m *Manager) compileAndExtractManifest(ctx context.Context, wasmPath string, config map[string]string) (*compiledPluginInfo, error) {
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("reading wasm file: %w", err)
	}

	// Compute SHA-256 hash
	hash := sha256.Sum256(wasmBytes)
	hashHex := hex.EncodeToString(hash[:])

	// Extract plugin name from path for logging
	pluginName := strings.TrimSuffix(filepath.Base(wasmPath), ".wasm")

	pluginManifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmData{Data: wasmBytes, Name: "main"},
		},
		Config:  config,
		Timeout: uint64(defaultTimeout.Milliseconds()),
	}
	extismConfig := extism.PluginConfig{
		EnableWasi:    true,
		RuntimeConfig: wazero.NewRuntimeConfig().WithCompilationCache(m.cache),
	}

	compiled, err := extism.NewCompiledPlugin(ctx, pluginManifest, extismConfig, stubHostFunctions())
	if err != nil {
		return nil, fmt.Errorf("compiling plugin: %w", err)
	}

	instance, err := compiled.Instance(ctx, extism.PluginInstanceConfig{})
	if err != nil {
		compiled.Close(ctx)
		return nil, fmt.Errorf("creating instance: %w", err)
	}
	defer instance.Close(ctx)
	instance.SetLogger(extismLogger(pluginName))

	exit, manifestBytes, err := instance.Call(manifestFunction, nil)
	if err != nil {
		compiled.Close(ctx)
		return nil, fmt.Errorf("calling manifest function: %w", err)
	}
	if exit != 0 {
		compiled.Close(ctx)
		return nil, fmt.Errorf("manifest function exited with code %d", exit)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		compiled.Close(ctx)
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	return &compiledPluginInfo{
		wasmBytes: wasmBytes,
		sha256:    hashHex,
		manifest:  &manifest,
		compiled:  compiled,
	}, nil
}

// ExtractManifest loads a wasm file, computes its SHA-256 hash, extracts the manifest,
// and immediately closes without full plugin initialization.
// This is a lightweight operation used for plugin discovery and change detection.
// The compilation is cached to speed up subsequent EnablePlugin calls.
func (m *Manager) ExtractManifest(wasmPath string) (*PluginMetadata, error) {
	if m.stopped.Load() {
		return nil, fmt.Errorf("manager is stopped")
	}

	info, err := m.compileAndExtractManifest(context.Background(), wasmPath, nil)
	if err != nil {
		return nil, err
	}
	defer info.compiled.Close(context.Background())

	return &PluginMetadata{
		Manifest: info.manifest,
		SHA256:   info.sha256,
	}, nil
}

// SyncPlugins scans the plugins folder and synchronizes with the database.
// It handles new, changed, and removed plugins by comparing SHA-256 hashes.
// - New plugins are added to DB as disabled
// - Changed plugins are updated in DB and disabled if they were enabled
// - Removed plugins are deleted from DB (after unloading if enabled)
func (m *Manager) SyncPlugins(ctx context.Context, folder string) error {
	if m.ds == nil {
		return fmt.Errorf("datastore not configured")
	}

	adminCtx := adminContext(ctx)

	// Read current plugins from folder
	entries, err := os.ReadDir(folder)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug(ctx, "Plugins folder does not exist", "folder", folder)
			return nil
		}
		return fmt.Errorf("reading plugins folder: %w", err)
	}

	// Build map of files in folder
	filesOnDisk := make(map[string]string) // name -> path
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".wasm") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".wasm")
		filesOnDisk[name] = filepath.Join(folder, entry.Name())
	}

	// Get all plugins from DB
	repo := m.ds.Plugin(adminCtx)
	dbPlugins, err := repo.GetAll()
	if err != nil {
		return fmt.Errorf("reading plugins from DB: %w", err)
	}
	pluginsInDB := make(map[string]*model.Plugin)
	for i := range dbPlugins {
		pluginsInDB[dbPlugins[i].ID] = &dbPlugins[i]
	}

	now := time.Now()

	// Process files on disk
	for name, path := range filesOnDisk {
		dbPlugin, exists := pluginsInDB[name]

		// Compute SHA256 first (lightweight operation) to check if plugin changed
		sha256Hash, err := computeFileSHA256(path)
		if err != nil {
			log.Error(ctx, "Failed to compute SHA256 for plugin", "plugin", name, "path", path, err)
			continue
		}

		// If plugin exists in DB with same hash, skip full manifest extraction
		if exists && dbPlugin.SHA256 == sha256Hash {
			// Plugin unchanged - just update path in case folder moved
			if dbPlugin.Path != path {
				dbPlugin.Path = path
				dbPlugin.UpdatedAt = now
				if err := repo.Put(dbPlugin); err != nil {
					log.Error(ctx, "Failed to update plugin path in DB", "plugin", name, err)
				}
			}
			delete(pluginsInDB, name)
			continue
		}

		// Plugin is new or changed - need full manifest extraction
		metadata, err := m.ExtractManifest(path)
		if err != nil {
			log.Error(ctx, "Failed to extract manifest from plugin", "plugin", name, "path", path, err)
			// Store error in DB if plugin exists
			if exists {
				dbPlugin.LastError = err.Error()
				dbPlugin.UpdatedAt = now
				if dbPlugin.Enabled {
					// Unload broken plugin
					if unloadErr := m.UnloadPlugin(name); unloadErr != nil {
						log.Debug(ctx, "Plugin not loaded", "plugin", name)
					}
					dbPlugin.Enabled = false
				}
				if putErr := repo.Put(dbPlugin); putErr != nil {
					log.Error(ctx, "Failed to update plugin in DB", "plugin", name, err)
				}
			}
			delete(pluginsInDB, name)
			continue
		}

		if !exists {
			// New plugin - add to DB as disabled
			if err := m.addPluginToDB(ctx, repo, name, path, metadata); err != nil {
				log.Error(ctx, "Failed to add plugin to DB", "plugin", name, err)
			}
		} else {
			// Plugin changed - update DB
			if err := m.updatePluginInDB(ctx, repo, dbPlugin, path, metadata); err != nil {
				log.Error(ctx, "Failed to update plugin in DB", "plugin", name, err)
			}
		}
		// Mark as processed
		delete(pluginsInDB, name)
	}

	// Remove plugins no longer on disk
	for _, dbPlugin := range pluginsInDB {
		if err := m.removePluginFromDB(ctx, repo, dbPlugin); err != nil {
			log.Error(ctx, "Failed to delete plugin from DB", "plugin", dbPlugin.ID, err)
		}
	}

	return nil
}

// loadEnabledPlugins loads all enabled plugins from the database.
func (m *Manager) loadEnabledPlugins(ctx context.Context) error {
	if m.ds == nil {
		return fmt.Errorf("datastore not configured")
	}

	adminCtx := adminContext(ctx)
	repo := m.ds.Plugin(adminCtx)

	plugins, err := repo.GetAll()
	if err != nil {
		return fmt.Errorf("reading plugins from DB: %w", err)
	}

	g := errgroup.Group{}
	g.SetLimit(maxPluginLoadConcurrency)

	for _, p := range plugins {
		if !p.Enabled {
			continue
		}

		plugin := p // Capture for goroutine
		g.Go(func() error {
			start := time.Now()
			log.Debug(ctx, "Loading enabled plugin", "plugin", plugin.ID, "path", plugin.Path)

			// Panic recovery
			defer func() {
				if r := recover(); r != nil {
					log.Error(ctx, "Panic while loading plugin", "plugin", plugin.ID, "panic", r)
				}
			}()

			if err := m.loadPluginWithConfig(plugin.ID, plugin.Path, plugin.Config); err != nil {
				// Store error in DB
				plugin.LastError = err.Error()
				plugin.Enabled = false
				plugin.UpdatedAt = time.Now()
				if putErr := repo.Put(&plugin); putErr != nil {
					log.Error(ctx, "Failed to update plugin error in DB", "plugin", plugin.ID, putErr)
				}
				log.Error(ctx, "Failed to load plugin", "plugin", plugin.ID, err)
				return nil
			}

			// Clear any previous error
			if plugin.LastError != "" {
				plugin.LastError = ""
				plugin.UpdatedAt = time.Now()
				if putErr := repo.Put(&plugin); putErr != nil {
					log.Error(ctx, "Failed to clear plugin error in DB", "plugin", plugin.ID, putErr)
				}
			}

			m.mu.RLock()
			loadedPlugin := m.plugins[plugin.ID]
			m.mu.RUnlock()
			if loadedPlugin != nil {
				log.Info(ctx, "Loaded plugin", "plugin", plugin.ID, "manifest", loadedPlugin.manifest.Name,
					"capabilities", loadedPlugin.capabilities, "duration", time.Since(start))
			}
			return nil
		})
	}

	return g.Wait()
}

// loadPluginWithConfig loads a plugin with configuration from DB.
func (m *Manager) loadPluginWithConfig(name, wasmPath, configJSON string) error {
	if m.stopped.Load() {
		return fmt.Errorf("manager is stopped")
	}

	// Track this operation
	m.loadWg.Add(1)
	defer m.loadWg.Done()

	if m.stopped.Load() {
		return fmt.Errorf("manager is stopped")
	}

	// Parse config from JSON
	var pluginConfig map[string]string
	if configJSON != "" {
		if err := json.Unmarshal([]byte(configJSON), &pluginConfig); err != nil {
			return fmt.Errorf("parsing plugin config: %w", err)
		}
	}

	// Compile and extract manifest using shared helper
	info, err := m.compileAndExtractManifest(m.ctx, wasmPath, pluginConfig)
	if err != nil {
		return err
	}

	// Create instance to detect capabilities
	instance, err := info.compiled.Instance(m.ctx, extism.PluginInstanceConfig{})
	if err != nil {
		info.compiled.Close(m.ctx)
		return fmt.Errorf("creating instance: %w", err)
	}
	instance.SetLogger(extismLogger(name))
	capabilities := detectCapabilities(instance)
	instance.Close(m.ctx)

	// Build host functions based on permissions
	var hostFunctions []extism.HostFunction
	var closers []io.Closer

	// Build extism manifest for potential recompilation
	pluginManifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmData{Data: info.wasmBytes, Name: "main"},
		},
		Config:  pluginConfig,
		Timeout: uint64(defaultTimeout.Milliseconds()),
	}

	if hosts := info.manifest.AllowedHosts(); len(hosts) > 0 {
		pluginManifest.AllowedHosts = hosts
	}

	// Register host functions based on permissions using table-driven approach
	svcCtx := &serviceContext{
		pluginName:  name,
		manager:     m,
		permissions: info.manifest.Permissions,
	}
	for _, entry := range hostServices {
		if entry.hasPermission(info.manifest.Permissions) {
			funcs, closer := entry.create(svcCtx)
			hostFunctions = append(hostFunctions, funcs...)
			if closer != nil {
				closers = append(closers, closer)
			}
		}
	}

	// Check if the plugin needs to be recompiled with real host functions
	compiled := info.compiled
	needsRecompile := len(pluginManifest.AllowedHosts) > 0 || len(hostFunctions) > 0

	// Recompile if needed. It is actually not a "recompile" since the first compilation
	// should be cached by wazero. We just need to do it this way to provide the real host functions.
	if needsRecompile {
		log.Trace(m.ctx, "Recompiling plugin with host functions", "plugin", name)
		info.compiled.Close(m.ctx)
		extismConfig := extism.PluginConfig{
			EnableWasi:    true,
			RuntimeConfig: wazero.NewRuntimeConfig().WithCompilationCache(m.cache),
		}
		compiled, err = extism.NewCompiledPlugin(m.ctx, pluginManifest, extismConfig, hostFunctions)
		if err != nil {
			return err
		}
	}

	m.mu.Lock()
	m.plugins[name] = &plugin{
		name:         name,
		path:         wasmPath,
		manifest:     info.manifest,
		compiled:     compiled,
		capabilities: capabilities,
		closers:      closers,
	}
	m.mu.Unlock()

	// Call plugin init function
	callPluginInit(m.ctx, m.plugins[name])

	return nil
}

// EnablePlugin enables a plugin by loading it and updating the DB.
// Returns an error if the plugin is not found in DB or fails to load.
func (m *Manager) EnablePlugin(ctx context.Context, id string) error {
	if m.ds == nil {
		return fmt.Errorf("datastore not configured")
	}

	adminCtx := adminContext(ctx)
	repo := m.ds.Plugin(adminCtx)

	plugin, err := repo.Get(id)
	if err != nil {
		return fmt.Errorf("getting plugin from DB: %w", err)
	}

	if plugin.Enabled {
		return nil // Already enabled
	}

	// Try to load the plugin
	if err := m.loadPluginWithConfig(plugin.ID, plugin.Path, plugin.Config); err != nil {
		// Store error and return
		plugin.LastError = err.Error()
		plugin.UpdatedAt = time.Now()
		_ = repo.Put(plugin)
		return fmt.Errorf("loading plugin: %w", err)
	}

	// Update DB
	plugin.Enabled = true
	plugin.LastError = ""
	plugin.UpdatedAt = time.Now()
	if err := repo.Put(plugin); err != nil {
		// Unload since we couldn't update DB
		_ = m.UnloadPlugin(id)
		return fmt.Errorf("updating plugin in DB: %w", err)
	}

	log.Info(ctx, "Enabled plugin", "plugin", id)
	return nil
}

// DisablePlugin disables a plugin by unloading it and updating the DB.
// Returns an error if the plugin is not found in DB.
func (m *Manager) DisablePlugin(ctx context.Context, id string) error {
	if m.ds == nil {
		return fmt.Errorf("datastore not configured")
	}

	adminCtx := adminContext(ctx)
	repo := m.ds.Plugin(adminCtx)

	plugin, err := repo.Get(id)
	if err != nil {
		return fmt.Errorf("getting plugin from DB: %w", err)
	}

	if !plugin.Enabled {
		return nil // Already disabled
	}

	// Unload the plugin
	if err := m.UnloadPlugin(id); err != nil {
		log.Debug(ctx, "Plugin was not loaded", "plugin", id)
	}

	// Update DB
	plugin.Enabled = false
	plugin.UpdatedAt = time.Now()
	if err := repo.Put(plugin); err != nil {
		return fmt.Errorf("updating plugin in DB: %w", err)
	}

	log.Info(ctx, "Disabled plugin", "plugin", id)
	return nil
}

// UpdatePluginConfig updates the configuration for a plugin.
// If the plugin is enabled, it will be reloaded with the new config.
func (m *Manager) UpdatePluginConfig(ctx context.Context, id, configJSON string) error {
	if m.ds == nil {
		return fmt.Errorf("datastore not configured")
	}

	adminCtx := adminContext(ctx)
	repo := m.ds.Plugin(adminCtx)

	plugin, err := repo.Get(id)
	if err != nil {
		return fmt.Errorf("getting plugin from DB: %w", err)
	}

	wasEnabled := plugin.Enabled

	// Update config in DB
	plugin.Config = configJSON
	plugin.UpdatedAt = time.Now()
	if err := repo.Put(plugin); err != nil {
		return fmt.Errorf("updating plugin config in DB: %w", err)
	}

	// Reload if enabled
	if wasEnabled {
		if err := m.UnloadPlugin(id); err != nil {
			log.Debug(ctx, "Plugin was not loaded", "plugin", id)
		}
		if err := m.loadPluginWithConfig(plugin.ID, plugin.Path, configJSON); err != nil {
			plugin.LastError = err.Error()
			plugin.Enabled = false
			_ = repo.Put(plugin)
			return fmt.Errorf("reloading plugin with new config: %w", err)
		}
	}

	log.Info(ctx, "Updated plugin config", "plugin", id)
	return nil
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

	runtime.GC()
	log.Info(m.ctx, "Unloaded plugin", "plugin", name)
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

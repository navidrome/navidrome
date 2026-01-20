package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Masterminds/squirrel"
	extism "github.com/extism/go-sdk"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/rjeczalik/notify"
	"github.com/tetratelabs/wazero"
)

const (
	// defaultTimeout is the default timeout for plugin function calls
	defaultTimeout = 30 * time.Second

	// maxPluginLoadConcurrency is the maximum number of plugins that can be
	// compiled/loaded in parallel during startup
	maxPluginLoadConcurrency = 3
)

// SubsonicRouter is an http.Handler that serves Subsonic API requests.
type SubsonicRouter = http.Handler

// PluginMetricsRecorder is an interface for recording plugin metrics.
// This is satisfied by core/metrics.Metrics but defined here to avoid import cycles.
type PluginMetricsRecorder interface {
	RecordPluginRequest(ctx context.Context, plugin, method string, ok bool, elapsed int64)
}

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
	broker         events.Broker
	metrics        PluginMetricsRecorder
}

// GetManager returns a singleton instance of the plugin manager.
// The manager is not started automatically; call Start() to begin loading plugins.
func GetManager(ds model.DataStore, broker events.Broker, m PluginMetricsRecorder) *Manager {
	return singleton.GetInstance(func() *Manager {
		return &Manager{
			ds:      ds,
			broker:  broker,
			metrics: m,
			plugins: make(map[string]*plugin),
		}
	})
}

// sendPluginRefreshEvent broadcasts a refresh event for the plugin resource.
// This notifies connected UI clients that plugin data has changed.
func (m *Manager) sendPluginRefreshEvent(ctx context.Context, pluginIDs ...string) {
	if m.broker == nil {
		return
	}
	event := (&events.RefreshResource{}).With("plugin", pluginIDs...)
	m.broker.SendBroadcastMessage(ctx, event)
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
	if err := m.syncPlugins(ctx, folder); err != nil {
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

	// Build user ID map for fast lookups
	userIDMap := make(map[string]struct{})
	for _, id := range plugin.allowedUserIDs {
		userIDMap[id] = struct{}{}
	}

	// Create a new scrobbler adapter for this plugin with user authorization config
	return &ScrobblerPlugin{
		name:           plugin.name,
		plugin:         plugin,
		allowedUserIDs: plugin.allowedUserIDs,
		allUsers:       plugin.allUsers,
		userIDMap:      userIDMap,
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

	// Check permission gates before enabling
	if err := m.checkPermissionGates(plugin); err != nil {
		return err
	}

	// Try to load the plugin
	if err := m.loadPluginWithConfig(plugin); err != nil {
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
		_ = m.unloadPlugin(id)
		return fmt.Errorf("updating plugin in DB: %w", err)
	}

	log.Info(ctx, "Enabled plugin", "plugin", id)
	m.sendPluginRefreshEvent(ctx, id)
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
	if err := m.unloadPlugin(id); err != nil {
		log.Debug(ctx, "Plugin was not loaded", "plugin", id)
	}

	// Update DB
	plugin.Enabled = false
	plugin.UpdatedAt = time.Now()
	if err := repo.Put(plugin); err != nil {
		return fmt.Errorf("updating plugin in DB: %w", err)
	}

	log.Info(ctx, "Disabled plugin", "plugin", id)
	m.sendPluginRefreshEvent(ctx, id)
	return nil
}

// ValidatePluginConfig validates a config JSON string against the plugin's config schema.
// If the plugin has no config schema defined, it returns an error.
// Returns nil if validation passes, or an error describing the validation failure.
func (m *Manager) ValidatePluginConfig(ctx context.Context, id, configJSON string) error {
	if m.ds == nil {
		return fmt.Errorf("datastore not configured")
	}

	adminCtx := adminContext(ctx)
	repo := m.ds.Plugin(adminCtx)

	plugin, err := repo.Get(id)
	if err != nil {
		return fmt.Errorf("getting plugin from DB: %w", err)
	}

	manifest, err := readManifest(plugin.Path)
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	return ValidateConfig(manifest, configJSON)
}

// UpdatePluginConfig updates the configuration for a plugin.
// If the plugin is enabled, it will be reloaded with the new config.
func (m *Manager) UpdatePluginConfig(ctx context.Context, id, configJSON string) error {
	return m.updatePluginSettings(ctx, id, func(p *model.Plugin) {
		p.Config = configJSON
	})
}

// UpdatePluginUsers updates the users permission settings for a plugin.
// If the plugin is enabled, it will be reloaded with the new settings.
// If the plugin requires users permission and no users are configured (and allUsers is false),
// the plugin will be automatically disabled.
func (m *Manager) UpdatePluginUsers(ctx context.Context, id, usersJSON string, allUsers bool) error {
	return m.updatePluginSettings(ctx, id, func(p *model.Plugin) {
		p.Users = usersJSON
		p.AllUsers = allUsers
	})
}

// UpdatePluginLibraries updates the libraries permission settings for a plugin.
// If the plugin is enabled, it will be reloaded with the new settings.
// If the plugin requires library permission and no libraries are configured (and allLibraries is false),
// the plugin will be automatically disabled.
func (m *Manager) UpdatePluginLibraries(ctx context.Context, id, librariesJSON string, allLibraries bool) error {
	return m.updatePluginSettings(ctx, id, func(p *model.Plugin) {
		p.Libraries = librariesJSON
		p.AllLibraries = allLibraries
	})
}

// RescanPlugins triggers a manual rescan of the plugins folder.
// This synchronizes the database with the filesystem, discovering new plugins,
// updating changed ones, and removing deleted ones.
func (m *Manager) RescanPlugins(ctx context.Context) error {
	folder := conf.Server.Plugins.Folder
	if folder == "" {
		return fmt.Errorf("plugins folder not configured")
	}
	log.Info(ctx, "Manual plugin rescan requested", "folder", folder)
	return m.syncPlugins(ctx, folder)
}

// updatePluginSettings is a common implementation for updating plugin settings.
// The updateFn is called to apply the specific field updates to the plugin.
// If the plugin is enabled, it will be reloaded. If users permission is required
// but no longer satisfied, the plugin will be disabled.
func (m *Manager) updatePluginSettings(ctx context.Context, id string, updateFn func(*model.Plugin)) error {
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

	// Apply the specific updates
	updateFn(plugin)
	plugin.UpdatedAt = time.Now()

	// Check if plugin requires permission and if it's still satisfied
	shouldDisable := false
	disableReason := ""
	if wasEnabled {
		manifest, err := readManifest(plugin.Path)
		if err == nil && manifest.Permissions != nil {
			if manifest.Permissions.Users != nil && !hasValidUsersConfig(plugin.Users, plugin.AllUsers) {
				shouldDisable = true
				disableReason = "users permission removal"
			}
			if manifest.Permissions.Library != nil && !hasValidLibrariesConfig(plugin.Libraries, plugin.AllLibraries) {
				shouldDisable = true
				disableReason = "library permission removal"
			}
		}
	}

	if shouldDisable {
		// Disable the plugin since permission is no longer satisfied
		if err := m.unloadPlugin(id); err != nil {
			log.Debug(ctx, "Plugin was not loaded", "plugin", id)
		}
		plugin.Enabled = false
		if err := repo.Put(plugin); err != nil {
			return fmt.Errorf("updating plugin in DB: %w", err)
		}
		log.Info(ctx, "Disabled plugin due to "+disableReason, "plugin", id)
		m.sendPluginRefreshEvent(ctx, id)
		return nil
	}

	if err := repo.Put(plugin); err != nil {
		return fmt.Errorf("updating plugin in DB: %w", err)
	}

	// Reload if enabled
	if wasEnabled {
		if err := m.unloadPlugin(id); err != nil {
			log.Debug(ctx, "Plugin was not loaded", "plugin", id)
		}
		if err := m.loadPluginWithConfig(plugin); err != nil {
			plugin.LastError = err.Error()
			plugin.Enabled = false
			_ = repo.Put(plugin)
			return fmt.Errorf("reloading plugin: %w", err)
		}
	}

	log.Info(ctx, "Updated plugin settings", "plugin", id)
	m.sendPluginRefreshEvent(ctx, id)
	return nil
}

// unloadPlugin removes a plugin from the manager and closes its resources.
// Returns an error if the plugin is not found.
func (m *Manager) unloadPlugin(name string) error {
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

// UnloadDisabledPlugins checks for plugins that are disabled in the database
// but still loaded in memory, and unloads them. This is called after user or
// library deletion to clean up plugins that were auto-disabled due to
// permission loss.
func (m *Manager) UnloadDisabledPlugins(ctx context.Context) {
	if m.ds == nil {
		return
	}

	adminCtx := adminContext(ctx)
	repo := m.ds.Plugin(adminCtx)

	// Get all disabled plugins from the database
	plugins, err := repo.GetAll(model.QueryOptions{
		Filters: squirrel.Eq{"enabled": false},
	})
	if err != nil {
		log.Error(ctx, "Failed to get disabled plugins", err)
		return
	}

	// Check each disabled plugin and unload if still in memory
	var unloaded []string
	for _, p := range plugins {
		m.mu.RLock()
		_, loaded := m.plugins[p.ID]
		m.mu.RUnlock()

		if loaded {
			if err := m.unloadPlugin(p.ID); err != nil {
				log.Warn(ctx, "Failed to unload disabled plugin", "plugin", p.ID, err)
			} else {
				unloaded = append(unloaded, p.ID)
				log.Info(ctx, "Unloaded disabled plugin", "plugin", p.ID)
			}
		}
	}

	// Send refresh events for unloaded plugins
	if len(unloaded) > 0 {
		m.sendPluginRefreshEvent(ctx, unloaded...)
	}
}

// checkPermissionGates validates that all permission-based requirements are met
// before a plugin can be enabled. Returns an error if any gate condition fails.
func (m *Manager) checkPermissionGates(p *model.Plugin) error {
	// Parse manifest to check permissions
	manifest, err := readManifest(p.Path)
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	// Check users permission gate
	if manifest.Permissions != nil && manifest.Permissions.Users != nil {
		if !hasValidUsersConfig(p.Users, p.AllUsers) {
			return fmt.Errorf("users permission requires configuration: select users or enable 'all users' access")
		}
	}

	// Check library permission gate
	if manifest.Permissions != nil && manifest.Permissions.Library != nil {
		if !hasValidLibrariesConfig(p.Libraries, p.AllLibraries) {
			return fmt.Errorf("library permission requires configuration: select libraries or enable 'all libraries' access")
		}
	}

	return nil
}

// hasValidUsersConfig checks if a plugin has valid users configuration.
// Returns true if allUsers is true, or if usersJSON contains at least one user.
func hasValidUsersConfig(usersJSON string, allUsers bool) bool {
	if allUsers {
		return true
	}
	if usersJSON == "" {
		return false
	}
	var users []string
	if err := json.Unmarshal([]byte(usersJSON), &users); err != nil {
		return false
	}
	return len(users) > 0
}

// hasValidLibrariesConfig checks if a plugin has valid libraries configuration.
// Returns true if allLibraries is true, or if librariesJSON contains at least one library.
func hasValidLibrariesConfig(librariesJSON string, allLibraries bool) bool {
	if allLibraries {
		return true
	}
	if librariesJSON == "" {
		return false
	}
	var libraries []int
	if err := json.Unmarshal([]byte(librariesJSON), &libraries); err != nil {
		return false
	}
	return len(libraries) > 0
}

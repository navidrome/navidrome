package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	extism "github.com/extism/go-sdk"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/scheduler"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"golang.org/x/sync/errgroup"
)

// serviceContext provides dependencies needed by host service factories.
type serviceContext struct {
	pluginName       string
	manager          *Manager
	permissions      *Permissions
	config           map[string]string
	allowedUsers     []string // User IDs this plugin can access
	allUsers         bool     // If true, plugin can access all users
	allowedLibraries []int    // Library IDs this plugin can access
	allLibraries     bool     // If true, plugin can access all libraries
}

// baseCtx returns the manager's lifecycle context, for host services that
// outlive the plugin call that created them. It falls back to
// context.Background() when the manager was never started, which is the case
// for CLI commands (e.g. `navidrome plugin enable`) that load plugins without
// calling Start.
func (c *serviceContext) baseCtx() context.Context {
	if c.manager.ctx == nil {
		return context.Background()
	}
	return c.manager.ctx
}

// hostServiceEntry defines a host service for table-driven registration.
type hostServiceEntry struct {
	name          string
	hasPermission func(*Permissions) bool
	create        func(*serviceContext) ([]extism.HostFunction, io.Closer, error)
}

// hostServices defines all available host services.
// Adding a new host service only requires adding an entry here.
var hostServices = []hostServiceEntry{
	{
		name:          "Config",
		hasPermission: func(p *Permissions) bool { return true }, // Always available, no permission required
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer, error) {
			service := newConfigService(ctx.pluginName, ctx.config)
			return host.RegisterConfigHostFunctions(service), nil, nil
		},
	},
	{
		name:          "SubsonicAPI",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Subsonicapi != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer, error) {
			service := newSubsonicAPIService(ctx.pluginName, ctx.manager.subsonicRouter, ctx.manager.ds, newUserAccess(ctx.allowedUsers, ctx.allUsers))
			return host.RegisterSubsonicAPIHostFunctions(service), nil, nil
		},
	},
	{
		name:          "Scheduler",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Scheduler != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer, error) {
			service := newSchedulerService(ctx.pluginName, ctx.manager, scheduler.GetInstance())
			return host.RegisterSchedulerHostFunctions(service), service, nil
		},
	},
	{
		name:          "WebSocket",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Websocket != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer, error) {
			perm := ctx.permissions.Websocket
			service := newWebSocketService(ctx.baseCtx(), ctx.pluginName, ctx.manager, perm)
			return host.RegisterWebSocketHostFunctions(service), service, nil
		},
	},
	{
		name:          "Artwork",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Artwork != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer, error) {
			service := newArtworkService()
			return host.RegisterArtworkHostFunctions(service), nil, nil
		},
	},
	{
		name:          "Cache",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Cache != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer, error) {
			service := newCacheService(ctx.pluginName)
			return host.RegisterCacheHostFunctions(service), service, nil
		},
	},
	{
		name:          "Library",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Library != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer, error) {
			perm := ctx.permissions.Library
			service := newLibraryService(ctx.manager.ds, perm, ctx.allowedLibraries, ctx.allLibraries)
			return host.RegisterLibraryHostFunctions(service), nil, nil
		},
	},
	{
		name:          "KVStore",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Kvstore != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer, error) {
			perm := ctx.permissions.Kvstore
			service, err := newKVStoreService(ctx.baseCtx(), ctx.pluginName, perm)
			if err != nil {
				return nil, nil, err
			}
			return host.RegisterKVStoreHostFunctions(service), service, nil
		},
	},
	{
		name:          "Users",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Users != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer, error) {
			service := newUsersService(ctx.manager.ds, ctx.allowedUsers, ctx.allUsers)
			return host.RegisterUsersHostFunctions(service), nil, nil
		},
	},
	{
		name:          "Matcher",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Matcher != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer, error) {
			hasFilesystemPerm := ctx.permissions.Library != nil && ctx.permissions.Library.Filesystem
			service := newMatcherService(
				ctx.manager.ds, hasFilesystemPerm,
				newUserAccess(ctx.allowedUsers, ctx.allUsers),
				newLibraryAccess(ctx.allowedLibraries, ctx.allLibraries),
			)
			return host.RegisterMatcherHostFunctions(service), nil, nil
		},
	},
	{
		name:          "HTTP",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Http != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer, error) {
			perm := ctx.permissions.Http
			service := newHTTPService(ctx.pluginName, perm)
			return host.RegisterHTTPHostFunctions(service), nil, nil
		},
	},
	{
		name:          "Task",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Taskqueue != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer, error) {
			perm := ctx.permissions.Taskqueue
			maxConcurrency := int32(1)
			if perm.MaxConcurrency > 0 {
				maxConcurrency = int32(perm.MaxConcurrency)
			}
			service, err := newTaskQueueService(ctx.baseCtx(), ctx.pluginName, ctx.manager, maxConcurrency)
			if err != nil {
				return nil, nil, err
			}
			return host.RegisterTaskHostFunctions(service), service, nil
		},
	},
}

// extractManifest reads manifest from an .ndp package and computes its SHA-256 hash.
// This is a lightweight operation used for plugin discovery and change detection.
// Unlike the old implementation, this does NOT compile the wasm - just reads the manifest JSON.
func (m *Manager) extractManifest(ndpPath string) (*PluginMetadata, error) {
	if m.stopped.Load() {
		return nil, fmt.Errorf("manager is stopped")
	}

	manifest, err := ReadManifest(ndpPath)
	if err != nil {
		return nil, err
	}

	sha256Hash, err := ComputeFileSHA256(ndpPath)
	if err != nil {
		return nil, fmt.Errorf("computing hash: %w", err)
	}

	return &PluginMetadata{
		Manifest: manifest,
		SHA256:   sha256Hash,
	}, nil
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

			if err := m.loadPluginWithConfig(&plugin); err != nil {
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
// The p.Path should point to an .ndp package file.
func (m *Manager) loadPluginWithConfig(p *model.Plugin) error {
	// NewContext falls back to context.Background() when m.ctx is nil (unstarted manager)
	ctx := log.NewContext(m.ctx, "plugin", p.ID)

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
	pluginConfig, err := parsePluginConfig(p.Config)
	if err != nil {
		return err
	}

	// Parse users from JSON
	var allowedUsers []string
	if p.Users != "" {
		if err := json.Unmarshal([]byte(p.Users), &allowedUsers); err != nil {
			return fmt.Errorf("parsing plugin users: %w", err)
		}
	}

	// Parse libraries from JSON
	var allowedLibraries []int
	if p.Libraries != "" {
		if err := json.Unmarshal([]byte(p.Libraries), &allowedLibraries); err != nil {
			return fmt.Errorf("parsing plugin libraries: %w", err)
		}
	}

	// Open the .ndp package to get manifest and wasm bytes
	pkg, err := openPackage(p.Path)
	if err != nil {
		return fmt.Errorf("opening package: %w", err)
	}

	// Build extism manifest
	pluginManifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmData{Data: pkg.WasmBytes, Name: "main"},
		},
		Config:  pluginConfig,
		Timeout: uint64(defaultTimeout.Milliseconds()),
	}

	if pkg.Manifest.Permissions != nil && pkg.Manifest.Permissions.Http != nil {
		if hosts := pkg.Manifest.Permissions.Http.RequiredHosts; len(hosts) > 0 {
			pluginManifest.AllowedHosts = hosts
		}
	}

	// Configure filesystem access for library permission
	if pkg.Manifest.HasLibraryFilesystemPermission() {
		adminCtx := adminContext(ctx)
		libraries, err := m.ds.Library(adminCtx).GetAll()
		if err != nil {
			return fmt.Errorf("failed to get libraries for filesystem access: %w", err)
		}

		allowedPaths := buildAllowedPaths(ctx, libraries, allowedLibraries, p.AllLibraries, p.AllowWriteAccess)
		pluginManifest.AllowedPaths = allowedPaths
	}

	// Build host functions based on permissions from manifest
	var hostFunctions []extism.HostFunction
	var closers []io.Closer
	loaded := false
	// On success the closers are owned by the registered plugin; on any
	// failure past this point, close them so partially-created services
	// don't leak goroutines or file handles.
	defer func() {
		if !loaded {
			closeAll(closers)
		}
	}()

	svcCtx := &serviceContext{
		pluginName:       p.ID,
		manager:          m,
		permissions:      pkg.Manifest.Permissions,
		config:           pluginConfig,
		allowedUsers:     allowedUsers,
		allUsers:         p.AllUsers,
		allowedLibraries: allowedLibraries,
		allLibraries:     p.AllLibraries,
	}
	for _, entry := range hostServices {
		if entry.hasPermission(pkg.Manifest.Permissions) {
			funcs, closer, err := entry.create(svcCtx)
			if err != nil {
				return fmt.Errorf("creating %s service: %w", entry.name, err)
			}
			hostFunctions = append(hostFunctions, funcs...)
			if closer != nil {
				closers = append(closers, closer)
			}
		}
	}

	// Compile the plugin with all host functions
	runtimeConfig := wazero.NewRuntimeConfig().
		WithCompilationCache(m.cache).
		WithCloseOnContextDone(true)

	// Enable experimental threads if requested in manifest
	if pkg.Manifest.HasExperimentalThreads() {
		runtimeConfig = runtimeConfig.WithCoreFeatures(api.CoreFeaturesV2 | experimental.CoreFeaturesThreads)
		log.Debug(ctx, "Enabling experimental threads support")
	}

	extismConfig := extism.PluginConfig{
		EnableWasi:                true,
		RuntimeConfig:             runtimeConfig,
		EnableHttpResponseHeaders: true,
	}
	compiled, err := extism.NewCompiledPlugin(ctx, pluginManifest, extismConfig, hostFunctions)
	if err != nil {
		return fmt.Errorf("compiling plugin: %w", err)
	}

	// Create instance to detect capabilities
	instance, err := compiled.Instance(ctx, extism.PluginInstanceConfig{})
	if err != nil {
		compiled.Close(ctx)
		return fmt.Errorf("creating instance: %w", err)
	}
	instance.SetLogger(extismLogger(p.ID))
	capabilities := detectCapabilities(instance)
	instance.Close(ctx)

	// Validate manifest against detected capabilities
	if err := ValidateWithCapabilities(pkg.Manifest, capabilities); err != nil {
		compiled.Close(ctx)
		return fmt.Errorf("manifest validation: %w", err)
	}

	m.mu.Lock()
	m.plugins[p.ID] = &plugin{
		name:           p.ID,
		path:           p.Path,
		manifest:       pkg.Manifest,
		compiled:       compiled,
		capabilities:   capabilities,
		closers:        closers,
		metrics:        m.metrics,
		allowedUserIDs: allowedUsers,
		allUsers:       p.AllUsers,
		libraries:      newLibraryAccess(allowedLibraries, p.AllLibraries),
	}
	m.mu.Unlock()
	loaded = true

	// Call plugin init function
	callPluginInit(ctx, m.plugins[p.ID])

	return nil
}

// closeAll closes host service closers accumulated before a load failure,
// so partially-created services don't leak goroutines or file handles.
func closeAll(closers []io.Closer) {
	for _, c := range closers {
		_ = c.Close()
	}
}

// parsePluginConfig parses a JSON config string into a map of string values.
// For Extism, all config values must be strings, so non-string values are serialized as JSON.
func parsePluginConfig(configJSON string) (map[string]string, error) {
	if configJSON == "" {
		return nil, nil
	}
	var rawConfig map[string]any
	if err := json.Unmarshal([]byte(configJSON), &rawConfig); err != nil {
		return nil, fmt.Errorf("parsing plugin config: %w", err)
	}
	pluginConfig := make(map[string]string)
	for key, value := range rawConfig {
		switch v := value.(type) {
		case string:
			pluginConfig[key] = v
		default:
			// Serialize non-string values as JSON
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("serializing config value %q: %w", key, err)
			}
			pluginConfig[key] = string(jsonBytes)
		}
	}
	return pluginConfig, nil
}

// buildAllowedPaths constructs the extism AllowedPaths map for filesystem access.
// When allowWriteAccess is false (default), paths are prefixed with "ro:" for read-only.
// Only libraries that match the allowed set (or all libraries if allLibraries is true) are included.
func buildAllowedPaths(ctx context.Context, libraries model.Libraries, allowedLibraryIDs []int, allLibraries, allowWriteAccess bool) map[string]string {
	allowedLibrarySet := make(map[int]struct{}, len(allowedLibraryIDs))
	for _, id := range allowedLibraryIDs {
		allowedLibrarySet[id] = struct{}{}
	}
	allowedPaths := make(map[string]string)
	for _, lib := range libraries {
		_, allowed := allowedLibrarySet[lib.ID]
		if allLibraries || allowed {
			mountPoint := toPluginMountPoint(int32(lib.ID))
			hostPath := lib.Path
			if !allowWriteAccess {
				hostPath = "ro:" + hostPath
			}
			allowedPaths[hostPath] = mountPoint
			log.Trace(ctx, "Added library to allowed paths", "libraryID", lib.ID, "mountPoint", mountPoint, "writeAccess", allowWriteAccess, "hostPath", hostPath)
		}
	}
	if allowWriteAccess {
		log.Info(ctx, "Granting read-write filesystem access to libraries", "libraryCount", len(allowedPaths), "allLibraries", allLibraries)
	} else {
		log.Debug(ctx, "Granting read-only filesystem access to libraries", "libraryCount", len(allowedPaths), "allLibraries", allLibraries)
	}
	return allowedPaths
}

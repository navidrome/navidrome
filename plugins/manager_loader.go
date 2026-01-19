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

// hostServiceEntry defines a host service for table-driven registration.
type hostServiceEntry struct {
	name          string
	hasPermission func(*Permissions) bool
	create        func(*serviceContext) ([]extism.HostFunction, io.Closer)
}

// hostServices defines all available host services.
// Adding a new host service only requires adding an entry here.
var hostServices = []hostServiceEntry{
	{
		name:          "Config",
		hasPermission: func(p *Permissions) bool { return true }, // Always available, no permission required
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer) {
			service := newConfigService(ctx.pluginName, ctx.config)
			return host.RegisterConfigHostFunctions(service), nil
		},
	},
	{
		name:          "SubsonicAPI",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Subsonicapi != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer) {
			service := newSubsonicAPIService(ctx.pluginName, ctx.manager.subsonicRouter, ctx.manager.ds, ctx.allowedUsers, ctx.allUsers)
			return host.RegisterSubsonicAPIHostFunctions(service), nil
		},
	},
	{
		name:          "Scheduler",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Scheduler != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer) {
			service := newSchedulerService(ctx.pluginName, ctx.manager, scheduler.GetInstance())
			return host.RegisterSchedulerHostFunctions(service), service
		},
	},
	{
		name:          "WebSocket",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Websocket != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer) {
			perm := ctx.permissions.Websocket
			service := newWebSocketService(ctx.pluginName, ctx.manager, perm)
			return host.RegisterWebSocketHostFunctions(service), service
		},
	},
	{
		name:          "Artwork",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Artwork != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer) {
			service := newArtworkService()
			return host.RegisterArtworkHostFunctions(service), nil
		},
	},
	{
		name:          "Cache",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Cache != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer) {
			service := newCacheService(ctx.pluginName)
			return host.RegisterCacheHostFunctions(service), service
		},
	},
	{
		name:          "Library",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Library != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer) {
			perm := ctx.permissions.Library
			service := newLibraryService(ctx.manager.ds, perm, ctx.allowedLibraries, ctx.allLibraries)
			return host.RegisterLibraryHostFunctions(service), nil
		},
	},
	{
		name:          "KVStore",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Kvstore != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer) {
			perm := ctx.permissions.Kvstore
			service, err := newKVStoreService(ctx.pluginName, perm)
			if err != nil {
				log.Error("Failed to create KVStore service", "plugin", ctx.pluginName, err)
				return nil, nil
			}
			return host.RegisterKVStoreHostFunctions(service), service
		},
	},
	{
		name:          "Users",
		hasPermission: func(p *Permissions) bool { return p != nil && p.Users != nil },
		create: func(ctx *serviceContext) ([]extism.HostFunction, io.Closer) {
			service := newUsersService(ctx.manager.ds, ctx.allowedUsers, ctx.allUsers)
			return host.RegisterUsersHostFunctions(service), nil
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

	manifest, err := readManifest(ndpPath)
	if err != nil {
		return nil, err
	}

	sha256Hash, err := computeFileSHA256(ndpPath)
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
	if pkg.Manifest.Permissions != nil && pkg.Manifest.Permissions.Library != nil && pkg.Manifest.Permissions.Library.Filesystem {
		adminCtx := adminContext(m.ctx)
		libraries, err := m.ds.Library(adminCtx).GetAll()
		if err != nil {
			return fmt.Errorf("failed to get libraries for filesystem access: %w", err)
		}

		// Build a set of allowed library IDs for fast lookup
		allowedLibrarySet := make(map[int]struct{}, len(allowedLibraries))
		for _, id := range allowedLibraries {
			allowedLibrarySet[id] = struct{}{}
		}

		allowedPaths := make(map[string]string)
		for _, lib := range libraries {
			// Only mount if allLibraries is true or library is in the allowed list
			if p.AllLibraries {
				allowedPaths[lib.Path] = toPluginMountPoint(int32(lib.ID))
			} else if _, ok := allowedLibrarySet[lib.ID]; ok {
				allowedPaths[lib.Path] = toPluginMountPoint(int32(lib.ID))
			}
		}
		pluginManifest.AllowedPaths = allowedPaths
	}

	// Build host functions based on permissions from manifest
	var hostFunctions []extism.HostFunction
	var closers []io.Closer

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
			funcs, closer := entry.create(svcCtx)
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
		log.Debug(m.ctx, "Enabling experimental threads support", "plugin", p.ID)
	}

	extismConfig := extism.PluginConfig{
		EnableWasi:    true,
		RuntimeConfig: runtimeConfig,
	}
	compiled, err := extism.NewCompiledPlugin(m.ctx, pluginManifest, extismConfig, hostFunctions)
	if err != nil {
		return fmt.Errorf("compiling plugin: %w", err)
	}

	// Create instance to detect capabilities
	instance, err := compiled.Instance(m.ctx, extism.PluginInstanceConfig{})
	if err != nil {
		compiled.Close(m.ctx)
		return fmt.Errorf("creating instance: %w", err)
	}
	instance.SetLogger(extismLogger(p.ID))
	capabilities := detectCapabilities(instance)
	instance.Close(m.ctx)

	// Validate manifest against detected capabilities
	if err := ValidateWithCapabilities(pkg.Manifest, capabilities); err != nil {
		compiled.Close(m.ctx)
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
	}
	m.mu.Unlock()

	// Call plugin init function
	callPluginInit(m.ctx, m.plugins[p.ID])

	return nil
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

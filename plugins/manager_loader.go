package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	extism "github.com/extism/go-sdk"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/scheduler"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"golang.org/x/sync/errgroup"
)

// serviceContext provides dependencies needed by host service factories.
type serviceContext struct {
	pluginName  string
	manager     *Manager
	permissions *Permissions
	config      map[string]string
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
			perm := ctx.permissions.Subsonicapi
			service := newSubsonicAPIService(ctx.pluginName, ctx.manager.subsonicRouter, ctx.manager.ds, perm)
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
			service := newLibraryService(ctx.manager.ds, perm)
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
// The ndpPath should point to an .ndp package file.
func (m *Manager) loadPluginWithConfig(name, ndpPath, configJSON string) error {
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

	// Open the .ndp package to get manifest and wasm bytes
	pkg, err := openPackage(ndpPath)
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

	if hosts := pkg.Manifest.AllowedHosts(); len(hosts) > 0 {
		pluginManifest.AllowedHosts = hosts
	}

	// Configure filesystem access for library permission
	if pkg.Manifest.Permissions != nil && pkg.Manifest.Permissions.Library != nil && pkg.Manifest.Permissions.Library.Filesystem {
		adminCtx := adminContext(m.ctx)
		libraries, err := m.ds.Library(adminCtx).GetAll()
		if err != nil {
			return fmt.Errorf("failed to get libraries for filesystem access: %w", err)
		}

		allowedPaths := make(map[string]string)
		for _, lib := range libraries {
			allowedPaths[lib.Path] = toPluginMountPoint(int32(lib.ID))
		}
		pluginManifest.AllowedPaths = allowedPaths
	}

	// Build host functions based on permissions from manifest
	var hostFunctions []extism.HostFunction
	var closers []io.Closer

	svcCtx := &serviceContext{
		pluginName:  name,
		manager:     m,
		permissions: pkg.Manifest.Permissions,
		config:      pluginConfig,
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
		log.Debug(m.ctx, "Enabling experimental threads support", "plugin", name)
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
	instance.SetLogger(extismLogger(name))
	capabilities := detectCapabilities(instance)
	instance.Close(m.ctx)

	m.mu.Lock()
	m.plugins[name] = &plugin{
		name:         name,
		path:         ndpPath,
		manifest:     pkg.Manifest,
		compiled:     compiled,
		capabilities: capabilities,
		closers:      closers,
		metrics:      m.metrics,
	}
	m.mu.Unlock()

	// Call plugin init function
	callPluginInit(m.ctx, m.plugins[name])

	return nil
}

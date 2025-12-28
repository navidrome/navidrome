package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	extism "github.com/extism/go-sdk"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/scheduler"
	"github.com/tetratelabs/wazero"
	"golang.org/x/sync/errgroup"
)

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

// compiledPluginInfo holds the intermediate compilation result used by both
// extractManifest and loadPluginWithConfig.
type compiledPluginInfo struct {
	wasmBytes []byte
	sha256    string
	manifest  *Manifest
	compiled  *extism.CompiledPlugin
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

// extractManifest loads a wasm file, computes its SHA-256 hash, extracts the manifest,
// and immediately closes without full plugin initialization.
// This is a lightweight operation used for plugin discovery and change detection.
// The compilation is cached to speed up subsequent EnablePlugin calls.
func (m *Manager) extractManifest(wasmPath string) (*PluginMetadata, error) {
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

package plugins

import (
	"context"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/artwork"
	"github.com/navidrome/navidrome/plugins/host/cache"
	"github.com/navidrome/navidrome/plugins/host/config"
	"github.com/navidrome/navidrome/plugins/host/http"
	"github.com/navidrome/navidrome/plugins/host/scheduler"
	"github.com/navidrome/navidrome/plugins/host/websocket"
	"github.com/navidrome/navidrome/plugins/schema"
	"github.com/tetratelabs/wazero"
	wazeroapi "github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const maxParallelCompilations = 2 // Limit to 2 concurrent compilations

var (
	compileSemaphore = make(chan struct{}, maxParallelCompilations)
	compilationCache wazero.CompilationCache
	cacheOnce        sync.Once
	runtimePool      sync.Map // map[string]*pooledRuntime
)

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
			name        string
			isPermitted bool
			loadFunc    func() (map[string]wazeroapi.FunctionDefinition, error)
		}

		// List of all available host services with their permissions and loading functions. For each service, we check
		// if the plugin has the required permission before loading it.
		availableServices := []hostService{
			{"config", permissions.Config != nil, func() (map[string]wazeroapi.FunctionDefinition, error) {
				return loadHostLibrary[config.ConfigService](ctx, config.Instantiate, &configServiceImpl{pluginID: pluginID})
			}},
			{"scheduler", permissions.Scheduler != nil, func() (map[string]wazeroapi.FunctionDefinition, error) {
				return loadHostLibrary[scheduler.SchedulerService](ctx, scheduler.Instantiate, m.schedulerService.HostFunctions(pluginID))
			}},
			{"cache", permissions.Cache != nil, func() (map[string]wazeroapi.FunctionDefinition, error) {
				return loadHostLibrary[cache.CacheService](ctx, cache.Instantiate, newCacheService(pluginID))
			}},
			{"artwork", permissions.Artwork != nil, func() (map[string]wazeroapi.FunctionDefinition, error) {
				return loadHostLibrary[artwork.ArtworkService](ctx, artwork.Instantiate, &artworkServiceImpl{})
			}},
			{"http", permissions.Http != nil, func() (map[string]wazeroapi.FunctionDefinition, error) {
				httpPerms, err := parseHTTPPermissions(permissions.Http)
				if err != nil {
					return nil, fmt.Errorf("invalid http permissions for plugin %s: %w", pluginID, err)
				}
				return loadHostLibrary[http.HttpService](ctx, http.Instantiate, &httpServiceImpl{
					pluginID:    pluginID,
					permissions: httpPerms,
				})
			}},
			{"websocket", permissions.Websocket != nil, func() (map[string]wazeroapi.FunctionDefinition, error) {
				wsPerms, err := parseWebSocketPermissions(permissions.Websocket)
				if err != nil {
					return nil, fmt.Errorf("invalid websocket permissions for plugin %s: %w", pluginID, err)
				}
				return loadHostLibrary[websocket.WebSocketService](ctx, websocket.Instantiate, m.websocketService.HostFunctions(pluginID, wsPerms))
			}},
		}

		// Load only permitted services
		var grantedPermissions []string
		var libraries []map[string]wazeroapi.FunctionDefinition
		for _, service := range availableServices {
			if service.isPermitted {
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
		if err := combineLibraries(ctx, r, libraries...); err != nil {
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

// purgeCacheBySize removes the oldest files in dir until its total size is
// lower than or equal to maxSize. maxSize should be a human-readable string
// like "10MB" or "200K". If parsing fails or maxSize is "0", the function is
// a no-op.
func purgeCacheBySize(dir, maxSize string) {
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
			log.Trace("Failed to access plugin cache entry", "path", path, err)
			return nil //nolint:nilerr
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			log.Trace("Failed to get file info for plugin cache entry", "path", path, err)
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
			log.Warn("Failed to traverse plugin cache directory", "path", dir, err)
		}
		return
	}

	log.Trace("Current plugin cache size", "path", dir, "size", humanize.Bytes(total), "sizeLimit", humanize.Bytes(sizeLimit))
	if total <= sizeLimit {
		return
	}

	log.Debug("Purging plugin cache", "path", dir, "sizeLimit", humanize.Bytes(sizeLimit), "currentSize", humanize.Bytes(total))
	sort.Slice(files, func(i, j int) bool { return files[i].mod < files[j].mod })
	for _, f := range files {
		if total <= sizeLimit {
			break
		}
		if err := os.Remove(f.path); err != nil {
			log.Warn("Failed to remove plugin cache entry", "path", f.path, "size", humanize.Bytes(f.size), err)
			continue
		}
		total -= f.size
		log.Debug("Removed plugin cache entry", "path", f.path, "size", humanize.Bytes(f.size), "time", time.UnixMilli(f.mod), "remainingSize", humanize.Bytes(total))

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

type pluginState struct {
	ready chan struct{}
	err   error
}

// getCompilationCache returns the global compilation cache, creating it if necessary
func getCompilationCache() (wazero.CompilationCache, error) {
	var err error
	cacheOnce.Do(func() {
		cacheDir := filepath.Join(conf.Server.CacheFolder, "plugins")
		purgeCacheBySize(cacheDir, conf.Server.Plugins.CacheSize)
		compilationCache, err = wazero.NewCompilationCacheWithDir(cacheDir)
	})
	return compilationCache, err
}

// newWazeroModuleConfig creates the correct ModuleConfig for plugins
func newWazeroModuleConfig() wazero.ModuleConfig {
	return wazero.NewModuleConfig().WithStartFunctions("_initialize").WithStderr(log.Writer())
}

// pluginCompilationTimeout returns the timeout for plugin compilation
func pluginCompilationTimeout() time.Duration {
	if conf.Server.DevPluginCompilationTimeout > 0 {
		return conf.Server.DevPluginCompilationTimeout
	}
	return time.Minute
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
func combineLibraries(ctx context.Context, r wazero.Runtime, libs ...map[string]wazeroapi.FunctionDefinition) error {
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

// Instance pool configuration
const (
	defaultMaxInstances = 8
	defaultInstanceTTL  = time.Minute
)

// pooledModule wraps a wazero Module and returns it to the pool when closed.
type pooledModule struct {
	wazeroapi.Module
	pool *wasmInstancePool[wazeroapi.Module]
}

func (m *pooledModule) Close(ctx context.Context) error {
	m.pool.Put(ctx, m.Module)
	return nil
}

func (m *pooledModule) CloseWithExitCode(ctx context.Context, exitCode uint32) error {
	m.pool.Put(ctx, m.Module)
	return nil
}

func (m *pooledModule) IsClosed() bool {
	return false
}

// pooledRuntime wraps wazero.Runtime and pools module instances per plugin.
type pooledRuntime struct {
	wazero.Runtime
	pluginID     string
	maxInstances int
	ttl          time.Duration

	once sync.Once
	pool *wasmInstancePool[wazeroapi.Module]

	mu     sync.Mutex
	active []wazeroapi.Module
}

func newPooledRuntime(r wazero.Runtime, pluginID string) *pooledRuntime {
	return &pooledRuntime{
		Runtime:      r,
		pluginID:     pluginID,
		maxInstances: defaultMaxInstances,
		ttl:          defaultInstanceTTL,
	}
}

func (r *pooledRuntime) initPool(code wazero.CompiledModule, config wazero.ModuleConfig) {
	r.once.Do(func() {
		r.pool = newWasmInstancePool[wazeroapi.Module](r.pluginID, r.maxInstances, r.ttl, func(ctx context.Context) (wazeroapi.Module, error) {
			log.Trace(ctx, "pooledRuntime: creating new module", "plugin", r.pluginID)
			return r.Runtime.InstantiateModule(ctx, code, config)
		})
	})
}

func (r *pooledRuntime) InstantiateModule(ctx context.Context, code wazero.CompiledModule, config wazero.ModuleConfig) (wazeroapi.Module, error) {
	r.initPool(code, config)
	mod, err := r.pool.Get(ctx)
	if err != nil {
		return nil, err
	}
	wrapped := &pooledModule{Module: mod, pool: r.pool}
	log.Trace(ctx, "pooledRuntime: created wrapper for module", "plugin", r.pluginID, "underlyingModuleID", fmt.Sprintf("%p", mod), "wrapperID", fmt.Sprintf("%p", wrapped))
	r.mu.Lock()
	r.active = append(r.active, wrapped)
	r.mu.Unlock()
	return wrapped, nil
}

// Close returns all active module instances to the pool without closing the runtime.
func (r *pooledRuntime) Close(ctx context.Context) error {
	r.mu.Lock()
	mods := r.active
	r.active = nil
	r.mu.Unlock()
	for _, m := range mods {
		_ = m.Close(ctx)
	}
	return nil
}

func (r *pooledRuntime) CloseWithExitCode(ctx context.Context, code uint32) error {
	return r.Close(ctx)
}

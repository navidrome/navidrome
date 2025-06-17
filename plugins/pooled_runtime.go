package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/tetratelabs/wazero"
	wazeroapi "github.com/tetratelabs/wazero/api"
)

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
	pluginName   string
	maxInstances int
	ttl          time.Duration

	once sync.Once
	pool *wasmInstancePool[wazeroapi.Module]

	mu     sync.Mutex
	active []wazeroapi.Module
}

func newPooledRuntime(r wazero.Runtime, pluginName string) *pooledRuntime {
	return &pooledRuntime{
		Runtime:      r,
		pluginName:   pluginName,
		maxInstances: defaultMaxInstances,
		ttl:          defaultInstanceTTL,
	}
}

func (r *pooledRuntime) initPool(code wazero.CompiledModule, config wazero.ModuleConfig) {
	r.once.Do(func() {
		r.pool = NewWasmInstancePool[wazeroapi.Module](r.pluginName, r.maxInstances, r.ttl, func(ctx context.Context) (wazeroapi.Module, error) {
			log.Trace(ctx, "pooledRuntime: creating new module", "plugin", r.pluginName)
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
	log.Trace(ctx, "pooledRuntime: created wrapper for module", "plugin", r.pluginName, "underlyingModuleID", fmt.Sprintf("%p", mod), "wrapperID", fmt.Sprintf("%p", wrapped))
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

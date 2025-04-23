package plugins

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/navidrome/navidrome/log"
)

type wasmBasePlugin[T any] struct {
	pool     *sync.Pool
	wasmPath string
	name     string
}

func (w *wasmBasePlugin[T]) getValidPooledInstance(ctx context.Context) (*pooledInstance, error) {
	v := w.pool.Get()
	if v == nil {
		log.Error(ctx, "wasmBasePlugin: sync.Pool returned nil instance", "plugin", w.name, "path", w.wasmPath)
		return nil, fmt.Errorf("wasmBasePlugin: sync.Pool returned nil instance for plugin %s", w.name)
	}
	pooled, ok := v.(*pooledInstance)
	if !ok || pooled == nil || pooled.instance == nil {
		log.Error(ctx, "wasmBasePlugin: pool returned invalid type or nil instance", "plugin", w.name, "path", w.wasmPath, "type", fmt.Sprintf("%T", v))
		if pooled != nil {
			pooled.cleanup.Stop()
		}
		if closer, canClose := v.(interface{ Close(context.Context) error }); canClose {
			_ = closer.Close(ctx)
		}
		return nil, fmt.Errorf("wasmBasePlugin: pool returned invalid instance for plugin %s", w.name)
	}
	return pooled, nil
}

func (w *wasmBasePlugin[T]) createPoolCleanupFunc(ctx context.Context, pooled *pooledInstance, closer func(context.Context) error, start time.Time, methodName string, isNotFound func(error) bool) func(error) {
	return func(err error) {
		if err == nil || isNotFound(err) {
			w.pool.Put(pooled)
			log.Trace(ctx, "wasmBasePlugin: returned instance to pool", "plugin", w.name, "instanceID", pooled.id, "method", methodName, "elapsed", time.Since(start), err)
		} else {
			pooled.cleanup.Stop()
			log.Trace(ctx, "wasmBasePlugin: stopped GC cleanup", "plugin", w.name, "instanceID", pooled.id, "method", methodName)
			if closer != nil {
				_ = closer(ctx)
				log.Trace(ctx, "wasmBasePlugin: closed instance due to error", "plugin", w.name, "instanceID", pooled.id, "method", methodName, "elapsed", time.Since(start), err)
			} else {
				log.Error(ctx, "wasmBasePlugin: attempted to close instance due to error, but closer was nil", "plugin", w.name, "instanceID", pooled.id, "method", methodName, "elapsed", time.Since(start), err)
			}
		}
	}
}

// getInstance returns the plugin instance, a cleanup function, and error
func (w *wasmBasePlugin[T]) getInstance(ctx context.Context, methodName string, isNotFound func(error) bool) (T, func(error), error) {
	var zero T
	pooled, err := w.getValidPooledInstance(ctx)
	if err != nil {
		return zero, nil, err
	}
	log.Trace(ctx, "wasmBasePlugin: got instance from pool", "plugin", w.name, "instanceID", pooled.id, "method", methodName)
	inst := pooled.instance.(T)
	start := time.Now()
	closerInst := pooled.instance.(interface{ Close(context.Context) error })
	closeFn := w.createPoolCleanupFunc(ctx, pooled, closerInst.Close, start, methodName, isNotFound)
	return inst, closeFn, nil
}

func (w *wasmBasePlugin[T]) Close(ctx context.Context) error {
	for {
		v := w.pool.Get()
		if v == nil {
			break
		}
		pooled, ok := v.(*pooledInstance)
		if !ok || pooled == nil || pooled.instance == nil {
			log.Warn(ctx, "wasmBasePlugin: found invalid type or nil instance in pool during agent close", "plugin", w.name, "path", w.wasmPath, "type", fmt.Sprintf("%T", v))
			if pooled != nil {
				pooled.cleanup.Stop()
			}
			if closer, canClose := v.(interface{ Close(context.Context) error }); canClose {
				_ = closer.Close(ctx)
			}
			continue
		}
		pooled.cleanup.Stop()
		log.Trace(ctx, "wasmBasePlugin: stopped GC cleanup during agent close", "plugin", w.name, "instanceID", pooled.id)
		if closer, ok := pooled.instance.(interface{ Close(context.Context) error }); ok {
			_ = closer.Close(ctx)
			log.Trace(ctx, "wasmBasePlugin: closed instance during agent close", "plugin", w.name, "instanceID", pooled.id, "path", w.wasmPath)
		} else {
			log.Warn(ctx, "wasmBasePlugin: instance in pool during agent close does not implement Close", "plugin", w.name, "instanceID", pooled.id, "path", w.wasmPath)
		}
	}
	log.Trace(ctx, "wasmBasePlugin: agent closed", "plugin", w.name, "path", w.wasmPath)
	return nil
}

// Generic plugin pool creation
func newPluginPool[L any](loader L, wasmPath, pluginName string, loadFunc func(context.Context, L, string) (any, error)) *sync.Pool {
	return &sync.Pool{
		New: func() any {
			instanceID, _ := gonanoid.New(10)
			inst, err := loadFunc(context.Background(), loader, wasmPath)
			if err != nil {
				log.Error("pool: failed to load plugin instance", "plugin", pluginName, "path", wasmPath, err)
				return nil
			}
			log.Trace("pool: created new plugin instance", "plugin", pluginName, "instanceID", instanceID, "path", wasmPath)

			closer, ok := inst.(interface{ Close(context.Context) error })
			if !ok {
				return &pooledInstance{id: instanceID, instance: inst}
			}

			arg := cleanupArg{
				closer:     closer,
				pluginName: pluginName,
				wasmPath:   wasmPath,
				instanceID: instanceID,
			}
			cleanup := runtime.AddCleanup(&inst, cleanupFunc, arg)
			log.Trace("pool: registered GC cleanup for instance", "plugin", pluginName, "instanceID", instanceID, "path", wasmPath)

			return &pooledInstance{id: instanceID, instance: inst, cleanup: cleanup}
		},
	}
}

// pooledInstance holds a wasm instance and its associated cleanup handle
type pooledInstance struct {
	id       string
	instance any
	cleanup  runtime.Cleanup
}

// cleanupArg holds the necessary information for the GC cleanup function
type cleanupArg struct {
	closer     interface{ Close(context.Context) error }
	pluginName string
	wasmPath   string
	instanceID string
}

// cleanupFunc is the function registered with runtime.AddCleanup
func cleanupFunc(arg cleanupArg) {
	log.Trace("pool: GC cleanup closing instance", "plugin", arg.pluginName, "instanceID", arg.instanceID, "path", arg.wasmPath)
	if err := arg.closer.Close(context.Background()); err != nil {
		log.Error("pool: GC cleanup failed to close instance", "plugin", arg.pluginName, "instanceID", arg.instanceID, "path", arg.wasmPath, err)
	} else {
		log.Trace("pool: GC cleanup closed instance successfully", "plugin", arg.pluginName, "instanceID", arg.instanceID, "path", arg.wasmPath)
	}
}

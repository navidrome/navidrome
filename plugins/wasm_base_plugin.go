package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/navidrome/navidrome/log"
)

// LoaderFunc is a generic function type that loads a plugin
// This function is needed to bridge the type gap between the non-exported interface returned
// by the plugin loader's Load() method and the public interface we use in our code.
type LoaderFunc[S any, P any] func(ctx context.Context, loader P, path string) (S, error)

// wasmBasePlugin is a generic base implementation for WASM plugins.
// It requires two generic type parameters:
// - S: The service interface type that the plugin implements
// - P: The plugin loader type that creates plugin instances
//
// Note: Both loader and loadFunc are necessary due to a limitation in the code generated
// by protoc-gen-go-plugin. The plugin loaders (like ScrobblerPlugin) have Load() methods
// that return non-exported interface types (like Scrobbler), while our code works
// with the exported interfaces (like Scrobbler). The loadFunc bridges this gap.
type wasmBasePlugin[S any, P any] struct {
	pool     sync.Pool
	wasmPath string
	name     string
	loader   P
	loadFunc LoaderFunc[S, P]
}

func (w *wasmBasePlugin[S, P]) PluginName() string {
	return w.name
}

func (w *wasmBasePlugin[S, P]) GetInstance(ctx context.Context) (any, func(), error) {
	instance, closeFn, err := w.getInstance(ctx, "<none>")
	return instance, func() {
		closeFn(nil)
	}, err
}

func (w *wasmBasePlugin[S, P]) initPool(ctx context.Context) {
	if w.pool.New != nil {
		return
	}
	w.pool.New = func() any {
		inst, err := w.loadFunc(ctx, w.loader, w.wasmPath)
		if err != nil {
			log.Error(ctx, "wasmBasePlugin: failed to create instance for pool", "plugin", w.name, err)
			return nil
		}
		instanceID := fmt.Sprintf("%p", inst)
		log.Trace(ctx, "wasmBasePlugin: created new instance for pool", "plugin", w.name, "instanceID", instanceID)
		return inst
	}
}

// getInstance returns a new plugin instance, a cleanup function, and error
func (w *wasmBasePlugin[S, P]) getInstance(ctx context.Context, methodName string) (S, func(error), error) {
	w.initPool(ctx)

	inst := w.pool.Get().(S)
	instanceID := fmt.Sprintf("%p", inst)
	log.Trace(ctx, "wasmBasePlugin: got instance from pool", "plugin", w.name, "instanceID", instanceID, "method", methodName)
	return inst, func(opErr error) {
		if opErr == nil {
			log.Trace(ctx, "wasmBasePlugin: returning instance to pool", "plugin", w.name, "instanceID", instanceID, "method", methodName)
			w.pool.Put(inst)
			return
		}
		log.Error(ctx, "wasmBasePlugin: error in method, closing instance", "plugin", w.name, "instanceID", instanceID, "method", methodName, opErr)
		start := time.Now()
		if closer, ok := any(inst).(interface{ Close(context.Context) error }); ok {
			if closeErr := closer.Close(ctx); closeErr != nil {
				log.Error(ctx, "wasmBasePlugin: error closing instance", "plugin", w.name, "instanceID", instanceID, "method", methodName, "elapsed", time.Since(start), "closeErr", closeErr)
			} else {
				log.Trace(ctx, "wasmBasePlugin: closed instance", "plugin", w.name, "instanceID", instanceID, "method", methodName, "elapsed", time.Since(start), "closeErr", closeErr)
			}
		}
	}, nil
}

func callMethod[S any, R any](ctx context.Context, w wasmPlugin[S], methodName string, fn func(inst S) (R, error)) (R, error) {
	inst, done, err := w.getInstance(ctx, methodName)
	var r R
	if err != nil {
		return r, err
	}
	defer func() { done(err) }()
	r, err = fn(inst)
	if em, ok := any(w).(errorMapper); ok {
		return r, em.mapError(err)
	}
	return r, err
}

type wasmPlugin[S any] interface {
	getInstance(ctx context.Context, methodName string) (S, func(error), error)
}

type errorMapper interface {
	mapError(err error) error
}

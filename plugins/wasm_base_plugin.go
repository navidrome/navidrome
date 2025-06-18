package plugins

import (
	"context"
	"fmt"
	"time"

	"github.com/navidrome/navidrome/log"
)

// LoaderFunc is a generic function type that loads a plugin instance.
type loaderFunc[S any, P any] func(ctx context.Context, loader P, path string) (S, error)

// wasmBasePlugin is a generic base implementation for WASM plugins.
// S is the service interface type and P is the plugin loader type.
type wasmBasePlugin[S any, P any] struct {
	wasmPath   string
	name       string
	capability string
	loader     P
	loadFunc   loaderFunc[S, P]
}

func (w *wasmBasePlugin[S, P]) PluginName() string {
	return w.name
}

func (w *wasmBasePlugin[S, P]) ServiceType() string {
	return w.capability
}

func (w *wasmBasePlugin[S, P]) Instantiate(ctx context.Context) (any, func(), error) {
	return w.getInstance(ctx, "<none>")
}

func (w *wasmBasePlugin[S, P]) serviceName() string {
	return w.name + "_" + w.capability
}

// getInstance loads a new plugin instance and returns a cleanup function.
func (w *wasmBasePlugin[S, P]) getInstance(ctx context.Context, methodName string) (S, func(), error) {
	start := time.Now()
	inst, err := w.loadFunc(ctx, w.loader, w.wasmPath)
	if err != nil {
		var zero S
		return zero, func() {}, fmt.Errorf("wasmBasePlugin: failed to load instance for %s: %w", w.serviceName(), err)
	}
	instanceID := getInstanceID(inst)
	log.Trace(ctx, "wasmBasePlugin: loaded instance", "plugin", w.serviceName(), "instanceID", instanceID, "method", methodName, "elapsed", time.Since(start))
	return inst, func() {
		if closer, ok := any(inst).(interface{ Close(context.Context) error }); ok {
			_ = closer.Close(ctx)
		}
	}, nil
}

type wasmPlugin[S any] interface {
	getInstance(ctx context.Context, methodName string) (S, func(), error)
}

type errorMapper interface {
	mapError(err error) error
}

func callMethod[S any, R any](ctx context.Context, w wasmPlugin[S], methodName string, fn func(inst S) (R, error)) (R, error) {
	inst, done, err := w.getInstance(ctx, methodName)
	var r R
	if err != nil {
		return r, err
	}
	defer done()
	r, err = fn(inst)
	if em, ok := any(w).(errorMapper); ok {
		return r, em.mapError(err)
	}
	return r, err
}

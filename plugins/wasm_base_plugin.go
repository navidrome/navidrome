package plugins

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/id"
)

// newWasmBasePlugin creates a new instance of wasmBasePlugin with the required parameters.
func newWasmBasePlugin[S any, P any](wasmPath, id, capability string, m metrics.Metrics, loader P, loadFunc loaderFunc[S, P]) *wasmBasePlugin[S, P] {
	return &wasmBasePlugin[S, P]{
		wasmPath:   wasmPath,
		id:         id,
		capability: capability,
		loader:     loader,
		loadFunc:   loadFunc,
		metrics:    m,
	}
}

// LoaderFunc is a generic function type that loads a plugin instance.
type loaderFunc[S any, P any] func(ctx context.Context, loader P, path string) (S, error)

// wasmBasePlugin is a generic base implementation for WASM plugins.
// S is the service interface type and P is the plugin loader type.
type wasmBasePlugin[S any, P any] struct {
	wasmPath   string
	id         string
	capability string
	loader     P
	loadFunc   loaderFunc[S, P]
	metrics    metrics.Metrics
}

func (w *wasmBasePlugin[S, P]) PluginID() string {
	return w.id
}

func (w *wasmBasePlugin[S, P]) Instantiate(ctx context.Context) (any, func(), error) {
	return w.getInstance(ctx, "<none>")
}

func (w *wasmBasePlugin[S, P]) serviceName() string {
	return w.id + "_" + w.capability
}

func (w *wasmBasePlugin[S, P]) getMetrics() metrics.Metrics {
	return w.metrics
}

// getInstance loads a new plugin instance and returns a cleanup function.
func (w *wasmBasePlugin[S, P]) getInstance(ctx context.Context, methodName string) (S, func(), error) {
	start := time.Now()
	// Add context metadata for tracing
	ctx = log.NewContext(ctx, "capability", w.serviceName(), "method", methodName)

	inst, err := w.loadFunc(ctx, w.loader, w.wasmPath)
	if err != nil {
		var zero S
		return zero, func() {}, fmt.Errorf("wasmBasePlugin: failed to load instance for %s: %w", w.serviceName(), err)
	}
	// Add context metadata for tracing
	ctx = log.NewContext(ctx, "instanceID", getInstanceID(inst))
	log.Trace(ctx, "wasmBasePlugin: loaded instance", "elapsed", time.Since(start))
	return inst, func() {
		log.Trace(ctx, "wasmBasePlugin: finished using instance", "elapsed", time.Since(start))
		if closer, ok := any(inst).(interface{ Close(context.Context) error }); ok {
			_ = closer.Close(ctx)
		}
	}, nil
}

type wasmPlugin[S any] interface {
	PluginID() string
	getInstance(ctx context.Context, methodName string) (S, func(), error)
	getMetrics() metrics.Metrics
}

type errorMapper interface {
	mapError(err error) error
}

func callMethod[S any, R any](ctx context.Context, w wasmPlugin[S], methodName string, fn func(inst S) (R, error)) (R, error) {
	// Add a unique call ID to the context for tracing
	ctx = log.NewContext(ctx, "callID", id.NewRandom())

	inst, done, err := w.getInstance(ctx, methodName)
	var r R
	if err != nil {
		return r, err
	}
	start := time.Now()
	defer done()
	r, err = fn(inst)
	elapsed := time.Since(start)

	if em, ok := any(w).(errorMapper); ok {
		mappedErr := em.mapError(err)

		if !errors.Is(mappedErr, agents.ErrNotFound) {
			id := w.PluginID()
			isOk := mappedErr == nil
			metrics := w.getMetrics()
			if metrics != nil {
				metrics.RecordPluginRequest(ctx, id, methodName, isOk, elapsed.Milliseconds())
			}
			log.Trace(ctx, "callMethod", "plugin", id, "method", methodName, "ok", isOk, elapsed)
		}

		return r, mappedErr
	}
	return r, err
}

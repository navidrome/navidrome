package plugins

import (
	"context"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/navidrome/navidrome/log"
)

type wasmBasePlugin[T any] struct {
	wasmPath string
	name     string
	loader   any
	loadFunc func(context.Context, any, string) (T, error)
}

// getInstance returns a new plugin instance, a cleanup function, and error
func (w *wasmBasePlugin[T]) getInstance(ctx context.Context, methodName string, isNotFound func(error) bool) (T, func(error), error) {
	var zero T
	instanceID, _ := gonanoid.New(10)
	inst, err := w.loadFunc(ctx, w.loader, w.wasmPath)
	if err != nil {
		log.Error(ctx, "wasmBasePlugin: failed to create instance", "plugin", w.name, "instanceID", instanceID, "method", methodName, "err", err)
		return zero, nil, err
	}
	log.Trace(ctx, "wasmBasePlugin: created new instance", "plugin", w.name, "instanceID", instanceID, "method", methodName)
	start := time.Now()
	closeFn := func(opErr error) {
		if closer, ok := any(inst).(interface{ Close(context.Context) error }); ok {
			if closeErr := closer.Close(ctx); closeErr != nil {
				log.Error(ctx, "wasmBasePlugin: error closing instance", "plugin", w.name, "instanceID", instanceID, "method", methodName, "elapsed", time.Since(start), "closeErr", closeErr, "opErr", opErr)
			} else {
				log.Trace(ctx, "wasmBasePlugin: closed instance", "plugin", w.name, "instanceID", instanceID, "method", methodName, "elapsed", time.Since(start), "opErr", opErr)
			}
		}
	}
	return inst, closeFn, nil
}

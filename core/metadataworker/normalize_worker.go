package metadataworker

import (
	"context"
)

type normalizeWorkerPool struct{}

var persistentNormalizeWorkers = &normalizeWorkerPool{}

// PersistentNormalizeWorkers returns the shared Rust FTS normalize entrypoint.
func PersistentNormalizeWorkers() *normalizeWorkerPool {
	return persistentNormalizeWorkers
}

func (p *normalizeWorkerPool) Normalize(ctx context.Context, values ...string) (string, error) {
	if len(values) == 0 {
		return "", nil
	}
	return normalizeFTSGRPC(ctx, values)
}

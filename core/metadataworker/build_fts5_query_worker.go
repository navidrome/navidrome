package metadataworker

import (
	"context"
)

type buildFTS5QueryResult struct {
	Query    string
	Degraded bool
}

type buildFTS5QueryWorkerPool struct{}

var persistentBuildFTS5QueryWorkers = &buildFTS5QueryWorkerPool{}

// PersistentBuildFTS5QueryWorkers returns the shared Rust FTS5 query builder entrypoint.
func PersistentBuildFTS5QueryWorkers() *buildFTS5QueryWorkerPool {
	return persistentBuildFTS5QueryWorkers
}

func (p *buildFTS5QueryWorkerPool) Build(ctx context.Context, query string) (buildFTS5QueryResult, error) {
	return buildFTS5QueryGRPC(ctx, query)
}

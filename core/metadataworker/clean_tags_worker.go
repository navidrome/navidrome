package metadataworker

import (
	"context"

	"github.com/navidrome/navidrome/conf"
)

type cleanTagsWorkerPool struct{}

var persistentCleanTagsWorkers = &cleanTagsWorkerPool{}

// PersistentCleanTagsWorkers returns the shared Rust tag-clean entrypoint.
func PersistentCleanTagsWorkers() *cleanTagsWorkerPool {
	return persistentCleanTagsWorkers
}

func (p *cleanTagsWorkerPool) Clean(ctx context.Context, filePath string, raw map[string][]string, mappings map[string]TagMappingExport) (map[string][]string, error) {
	if len(raw) == 0 {
		return map[string][]string{}, nil
	}
	return cleanTagsGRPC(ctx, filePath, raw, mappings, confArtistSplitExceptions())
}

func confArtistSplitExceptions() []string {
	return append([]string(nil), conf.Server.Scanner.ArtistSplitExceptions...)
}

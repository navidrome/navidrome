package metadataworker

import (
	"context"
	"fmt"
)

// MapMediaFileJSON runs the Rust map-media gRPC worker for tests and fake extractors.
func MapMediaFileJSON(path string, tags map[string][]string, lyricsJSON string) (string, error) {
	if _, err := Resolve(); err != nil {
		return "", fmt.Errorf("resolve metadata worker: %w", err)
	}
	if lyricsJSON == "" {
		lyricsJSON = "[]"
	}
	return mapMediaGRPC(context.Background(), path, tags, lyricsJSON, WorkerScanConfig{})
}

package metadataworker

import (
	"context"
	"fmt"
	"sync"

	"github.com/navidrome/navidrome/core/rustworker"
)

const maxLyricsInputBytes = 16 * 1024 * 1024

type lyricsWorkerPool struct{}

var persistentLyricsWorkers = &lyricsWorkerPool{}

// PersistentLyricsWorkers returns the shared Rust lyrics parser entrypoint.
func PersistentLyricsWorkers() *lyricsWorkerPool {
	return persistentLyricsWorkers
}

func (p *lyricsWorkerPool) Parse(ctx context.Context, suffix, lang string, contents []byte) (string, error) {
	return p.parse(ctx, suffix, lang, contents)
}

func (p *lyricsWorkerPool) parse(ctx context.Context, suffix, lang string, contents []byte) (string, error) {
	if len(contents) == 0 {
		return "[]", nil
	}
	if len(contents) > maxLyricsInputBytes {
		return "", fmt.Errorf("lyrics payload exceeds maximum size of %d bytes", maxLyricsInputBytes)
	}
	return parseLyricsGRPC(ctx, suffix, lang, contents)
}

var testBinaryOnce sync.Once

// EnsureTestBinary builds or locates navidrome-metadata for Go tests when
// ND_METADATAWORKERPATH is unset.
func EnsureTestBinary() error {
	var setupErr error
	testBinaryOnce.Do(func() {
		setupErr = rustworker.EnsureTestBinary(EnvPath, "navidrome-metadata", BinaryName())
	})
	return setupErr
}

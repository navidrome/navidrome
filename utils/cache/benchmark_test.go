package cache

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
)

type benchItem struct {
	key string
}

func (b *benchItem) Key() string { return b.key }

// setupBenchCache creates a file cache in a temp directory. Returns the cache and cleanup function.
func setupBenchCache(b *testing.B, cacheSize string, getReader ReadFunc) (*fileCache, func()) {
	b.Helper()
	tmpDir, err := os.MkdirTemp("", "bench-cache-*")
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(configtest.SetupConfig())
	conf.Server.CacheFolder = tmpDir

	fc := NewFileCache("bench", cacheSize, "bench", 0, getReader).(*fileCache)

	// Wait for cache to be ready
	for !fc.ready.Load() {
		runtime.Gosched() // Yield to allow background init goroutine to run
	}

	teardown := func() {
		os.RemoveAll(tmpDir)
	}
	return fc, teardown
}

func BenchmarkCacheWrite(b *testing.B) {
	// Simulate writing 50KB images (typical 300px JPEG)
	imageData := strings.Repeat("x", 50*1024)

	fc, cleanup := setupBenchCache(b, "100MB", func(ctx context.Context, item Item) (io.Reader, error) {
		return strings.NewReader(imageData), nil
	})
	defer cleanup()

	b.SetBytes(int64(len(imageData)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("write-bench-%d", i)
		s, err := fc.Get(context.Background(), &benchItem{key: key})
		if err != nil {
			b.Fatal(err)
		}
		_, _ = io.ReadAll(s)
		s.Close()
	}
}

func BenchmarkCacheRead(b *testing.B) {
	imageData := strings.Repeat("x", 50*1024)

	fc, cleanup := setupBenchCache(b, "100MB", func(ctx context.Context, item Item) (io.Reader, error) {
		return strings.NewReader(imageData), nil
	})
	defer cleanup()

	// Pre-populate cache
	item := &benchItem{key: "read-bench"}
	s, err := fc.Get(context.Background(), item)
	if err != nil {
		b.Fatal(err)
	}
	_, _ = io.ReadAll(s)
	s.Close()

	b.SetBytes(int64(len(imageData)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s, err := fc.Get(context.Background(), item)
		if err != nil {
			b.Fatal(err)
		}
		_, _ = io.ReadAll(s)
		s.Close()
	}
}

func BenchmarkConcurrentCacheRead(b *testing.B) {
	imageData := strings.Repeat("x", 50*1024)

	fc, cleanup := setupBenchCache(b, "100MB", func(ctx context.Context, item Item) (io.Reader, error) {
		return strings.NewReader(imageData), nil
	})
	defer cleanup()

	// Pre-populate cache
	item := &benchItem{key: "concurrent-read"}
	s, _ := fc.Get(context.Background(), item)
	_, _ = io.ReadAll(s)
	s.Close()

	concurrencyLevels := []int{1, 10, 50}
	for _, n := range concurrencyLevels {
		b.Run(fmt.Sprintf("goroutines_%d", n), func(b *testing.B) {
			b.SetBytes(int64(len(imageData)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				wg.Add(n)
				for g := 0; g < n; g++ {
					go func() {
						defer wg.Done()
						s, err := fc.Get(context.Background(), item)
						if err != nil {
							b.Error(err)
							return
						}
						_, _ = io.ReadAll(s)
						s.Close()
					}()
				}
				wg.Wait()
			}
		})
	}
}

func BenchmarkConcurrentCacheMiss(b *testing.B) {
	imageData := strings.Repeat("x", 50*1024)

	concurrencyLevels := []int{1, 10, 50}
	for _, n := range concurrencyLevels {
		b.Run(fmt.Sprintf("goroutines_%d", n), func(b *testing.B) {
			fc, cleanup := setupBenchCache(b, "100MB", func(ctx context.Context, item Item) (io.Reader, error) {
				return strings.NewReader(imageData), nil
			})
			defer cleanup()

			b.SetBytes(int64(len(imageData)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				wg.Add(n)
				// All goroutines request the SAME key (not yet cached)
				item := &benchItem{key: fmt.Sprintf("miss-%d", i)}
				for g := 0; g < n; g++ {
					go func() {
						defer wg.Done()
						s, err := fc.Get(context.Background(), item)
						if err != nil {
							b.Error(err)
							return
						}
						_, _ = io.ReadAll(s)
						s.Close()
					}()
				}
				wg.Wait()
			}
		})
	}
}

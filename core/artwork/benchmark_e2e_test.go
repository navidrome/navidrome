package artwork

import (
	"context"
	"fmt"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/cache"
)

// setupE2EBenchmark creates an artwork instance with a real album cover image on disk,
// backed by either a real file cache or disabled cache depending on cacheSize.
// Note: This benchmarks artwork.Get() directly (not the full HTTP handler), which covers
// the critical path (source selection, decode, resize, encode, cache). This is a deliberate
// spec deviation — the full HTTP round-trip benchmark requires significant infrastructure
// (DB, scanner, fake filesystem) and can be added later if HTTP overhead proves significant.
//
// Depends on fakeFolderRepo defined in reader_artist_test.go (same package, compiled together).
func setupE2EBenchmark(b *testing.B, cacheSize string) (Artwork, model.ArtworkID, func()) {
	b.Helper()
	cleanup := configtest.SetupConfig()
	b.Cleanup(cleanup)

	tmpDir, err := os.MkdirTemp("", "artwork-bench-*")
	if err != nil {
		b.Fatal(err)
	}

	// Create a realistic cover image on disk
	coverPath := filepath.Join(tmpDir, "cover.jpg")
	coverImg := generateGradientImage(1000, 1000)
	f, err := os.Create(coverPath)
	if err != nil {
		b.Fatal(err)
	}
	if err := jpeg.Encode(f, coverImg, &jpeg.Options{Quality: 90}); err != nil {
		f.Close()
		b.Fatal(err)
	}
	f.Close()

	// Configure cache
	conf.Server.ImageCacheSize = cacheSize
	conf.Server.CacheFolder = tmpDir
	conf.Server.CoverArtQuality = 75
	conf.Server.CoverArtPriority = "cover.*"

	// Set up mock data store with album pointing to our cover.
	// Set UpdatedAt so CoverArtID().LastUpdate is consistent across calls.
	album := model.Album{
		ID:        "bench-album-1",
		Name:      "Benchmark Album",
		FolderIDs: []string{"f1"},
		UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	folderRepo := &fakeFolderRepo{
		result: []model.Folder{{
			Path:       tmpDir,
			ImageFiles: []string{"cover.jpg"},
		}},
	}
	ds := &tests.MockDataStore{
		MockedTranscoding: &tests.MockTranscodingRepo{},
		MockedFolder:      folderRepo,
	}
	ds.Album(context.Background()).(*tests.MockAlbumRepo).SetData(model.Albums{album})

	artID := album.CoverArtID()

	imgCache := cache.NewFileCache("BenchImage", cacheSize, "bench-images", 0,
		func(ctx context.Context, arg cache.Item) (io.Reader, error) {
			r, _, err := arg.(artworkReader).Reader(ctx)
			return r, err
		})

	// Wait for cache init if enabled
	if cacheSize != "0" {
		for !imgCache.Available(context.Background()) && !imgCache.Disabled(context.Background()) {
			runtime.Gosched() // Yield to allow background init goroutine to run
		}
	}

	ffmpeg := tests.NewMockFFmpeg("fallback content")
	aw := NewArtwork(ds, imgCache, ffmpeg, nil)

	cleanupAll := func() {
		os.RemoveAll(tmpDir)
	}
	return aw, artID, cleanupAll
}

func BenchmarkArtworkGetE2E(b *testing.B) {
	cacheConfigs := []struct {
		name      string
		cacheSize string
	}{
		{"no_cache", "0"},
		{"with_cache", "100MB"},
	}
	sizes := []int{0, 300}

	for _, cc := range cacheConfigs {
		for _, size := range sizes {
			b.Run(fmt.Sprintf("%s/size_%d", cc.name, size), func(b *testing.B) {
				aw, artID, cleanup := setupE2EBenchmark(b, cc.cacheSize)
				defer cleanup()

				// Warm the cache on first call if cache is enabled
				if cc.cacheSize != "0" {
					r, _, err := aw.Get(context.Background(), artID, size, size > 0)
					if err != nil {
						b.Fatal(err)
					}
					_, _ = io.ReadAll(r)
					r.Close()
				}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					r, _, err := aw.Get(context.Background(), artID, size, size > 0)
					if err != nil {
						b.Fatal(err)
					}
					_, _ = io.ReadAll(r)
					r.Close()
				}
			})
		}
	}
}

func BenchmarkArtworkGetE2EConcurrent(b *testing.B) {
	cacheConfigs := []struct {
		name      string
		cacheSize string
	}{
		{"no_cache", "0"},
		{"with_cache", "100MB"},
	}
	concurrencyLevels := []int{10, 50}

	for _, cc := range cacheConfigs {
		for _, n := range concurrencyLevels {
			b.Run(fmt.Sprintf("%s/goroutines_%d", cc.name, n), func(b *testing.B) {
				aw, artID, cleanup := setupE2EBenchmark(b, cc.cacheSize)
				defer cleanup()

				// Warm cache
				if cc.cacheSize != "0" {
					r, _, _ := aw.Get(context.Background(), artID, 300, true)
					if r != nil {
						_, _ = io.ReadAll(r)
						r.Close()
					}
				}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var wg sync.WaitGroup
					wg.Add(n)
					for g := 0; g < n; g++ {
						go func() {
							defer wg.Done()
							r, _, err := aw.Get(context.Background(), artID, 300, true)
							if err != nil {
								b.Error(err)
								return
							}
							_, _ = io.ReadAll(r)
							r.Close()
						}()
					}
					wg.Wait()
				}
			})
		}
	}
}

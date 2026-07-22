package artwork

import (
	"context"
	"io"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
)

// soakCycles is deliberately >2000: this is a leak regression guard, not a
// performance benchmark, so it favors a stable signal over raw speed.
const soakCycles = 2200

// TestWorkerSoak drives processItem across a mix of sources (folder, embedded
// extraction, dangling refs) for many cycles, reading each acquired image back
// through ImageStore.Open, and asserts goroutines/heap plateau instead of
// growing unbounded. Skipped under -short.
func TestWorkerSoak(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping soak test in short mode")
	}
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	defer configtest.SetupConfig()()

	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	libRepo := &tests.MockLibraryRepo{}
	libRepo.SetData(model.Libraries{{ID: 0, Path: testFileLibPath(repoRoot)}})
	folderRepo := &fakeFolderRepo{result: []model.Folder{{
		Path:       "tests/fixtures/artist/an-album",
		ImageFiles: []string{"cover.jpg"},
	}}}
	ffm := tests.NewMockFFmpeg("")
	prov := &fakeExternalProvider{}
	artRepo := tests.CreateMockArtworkRepo()
	albumRepo := tests.CreateMockAlbumRepo()
	albumRepo.SetData(model.Albums{
		{ID: "al-folder", Name: "Folder Album", FolderIDs: []string{"f1"}},
		{ID: "al-embed", Name: "Embedded Album", EmbedArtPath: "tests/fixtures/artist/an-album/test.mp3", FolderIDs: []string{"f1"}},
	})
	ds := &tests.MockDataStore{
		MockedFolder:  folderRepo,
		MockedLibrary: libRepo,
		MockedArtwork: artRepo,
		MockedAlbum:   albumRepo,
	}
	store := NewImageStore(t.TempDir())
	deps := &workerDeps{ds: ds, store: store, prov: prov, ffmpeg: ffm}
	conf.Server.CoverArtPriority = "cover.jpg, embedded"

	// Dangling refs (al/ra ids the repos don't know about) mirror an entity
	// deleted after being enqueued; ds.Radio auto-provisions an empty mock repo.
	items := []model.ArtworkQueueItem{
		{ItemKind: "al", ItemID: "al-folder"},
		{ItemKind: "al", ItemID: "al-embed"},
		{ItemKind: "al", ItemID: "al-does-not-exist"},
		{ItemKind: "ra", ItemID: "ra-does-not-exist"},
	}

	fdCount := func() int {
		if runtime.GOOS != "linux" {
			return -1
		}
		entries, err := os.ReadDir("/proc/self/fd")
		if err != nil {
			return -1
		}
		return len(entries)
	}

	settleGoroutines := func() int {
		// Background goroutines (GC workers, etc.) can take a moment to wind down;
		// poll for two consecutive equal samples instead of trusting a single one.
		prev := -1
		for range 100 {
			runtime.GC()
			n := runtime.NumGoroutine()
			if n == prev {
				return n
			}
			prev = n
			time.Sleep(10 * time.Millisecond)
		}
		return prev
	}

	baselineGoroutines := settleGoroutines()
	baselineFDs := fdCount()

	var heapAt10Pct uint64
	start := time.Now()
	for i := range soakCycles {
		it := items[i%len(items)]
		out := processItem(context.Background(), deps, it)

		// "Serve-adjacent" read-back: exercise the Phase 2 surfaces a caller would
		// use after acquisition, not the old serving pipeline.
		if out == outcomeFound {
			ia, err := artRepo.GetItemArtwork(it.ItemKind, it.ItemID, model.ImageTypePrimary)
			if err != nil {
				t.Fatalf("cycle %d: GetItemArtwork: %v", i, err)
			}
			art, err := artRepo.GetImage(ia.Hash)
			if err != nil {
				t.Fatalf("cycle %d: GetImage: %v", i, err)
			}
			rc, err := store.Open(ia.Hash, art.Mime)
			switch {
			case err == nil:
				_, _ = io.Copy(io.Discard, rc)
				rc.Close()
			case os.IsNotExist(err):
				// Folder-backed art has no store file; that's expected.
			default:
				t.Fatalf("cycle %d: store.Open: %v", i, err)
			}
		}

		if i == soakCycles/10 {
			runtime.GC()
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			heapAt10Pct = ms.HeapAlloc
		}
	}
	elapsed := time.Since(start)

	finalGoroutines := settleGoroutines()
	finalFDs := fdCount()

	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	t.Logf("soak: cycles=%d elapsed=%s goroutines(baseline=%d final=%d) heap(10%%-mark=%d final=%d) fds(baseline=%d final=%d)",
		soakCycles, elapsed, baselineGoroutines, finalGoroutines, heapAt10Pct, ms.HeapAlloc, baselineFDs, finalFDs)

	if finalGoroutines > baselineGoroutines {
		t.Errorf("goroutine count grew: baseline=%d final=%d", baselineGoroutines, finalGoroutines)
	}
	if heapAt10Pct > 0 && ms.HeapAlloc > 2*heapAt10Pct {
		t.Errorf("heap did not plateau: 10%%-mark=%d final=%d (final > 2x 10%%-mark)", heapAt10Pct, ms.HeapAlloc)
	}
	if runtime.GOOS == "linux" && baselineFDs >= 0 && finalFDs > baselineFDs {
		t.Errorf("fd count grew: baseline=%d final=%d", baselineFDs, finalFDs)
	}
}

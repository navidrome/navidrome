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
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Deliberately >2000: a leak guard favors a stable signal over speed.
const soakCycles = 2200

var _ = Describe("Worker soak", func() {
	It("does not leak goroutines, heap, or fds over many acquisition cycles", func() {
		if testing.Short() {
			Skip("skipping soak test in short mode")
		}
		DeferCleanup(configtest.SetupConfig())

		repoRoot, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())

		libRepo := &tests.MockLibraryRepo{}
		libRepo.SetData(model.Libraries{{ID: 0, Path: testFileLibPath(repoRoot)}})
		folderRepo := &fakeFolderRepo{result: []model.Folder{{
			Path:       "tests/fixtures/artist/an-album",
			ImageFiles: []string{"cover.jpg"},
		}}}
		ffm := tests.NewMockFFmpeg("")
		ag := agents.GetAgents(&tests.MockDataStore{}, nil)
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
		store := NewImageStore(GinkgoT().TempDir())
		proc := &processor{ds: ds, store: store, resolver: newResolver(ds, ag, ffm, nil)}
		conf.Server.CoverArtPriority = "cover.jpg, embedded"

		// Dangling refs mirror an entity deleted after being enqueued.
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
			// Background goroutines wind down late, so poll for two consecutive equal samples.
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
			out, _ := proc.acquire(context.Background(), it)

			// Read-back exercises the surfaces a caller would use after acquisition.
			if out == outcomeFound {
				kind, _ := model.ParseKind(it.ItemKind)
				ia, err := artRepo.GetItemArtwork(kind, it.ItemID, model.ImageTypePrimary)
				Expect(err).ToNot(HaveOccurred(), "cycle %d: GetItemArtwork", i)
				art, err := artRepo.GetImage(ia.Hash)
				Expect(err).ToNot(HaveOccurred(), "cycle %d: GetImage", i)
				rc, err := store.Open(ia.Hash, art.Mime)
				switch {
				case err == nil:
					_, _ = io.Copy(io.Discard, rc)
					rc.Close()
				case os.IsNotExist(err):
					// Folder-backed art has no store file; that's expected.
				default:
					Expect(err).ToNot(HaveOccurred(), "cycle %d: store.Open", i)
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

		GinkgoWriter.Printf("soak: cycles=%d elapsed=%s goroutines(baseline=%d final=%d) heap(10%%-mark=%d final=%d) fds(baseline=%d final=%d)\n",
			soakCycles, elapsed, baselineGoroutines, finalGoroutines, heapAt10Pct, ms.HeapAlloc, baselineFDs, finalFDs)

		Expect(finalGoroutines).To(BeNumerically("<=", baselineGoroutines), "goroutine count grew: baseline=%d final=%d", baselineGoroutines, finalGoroutines)
		if heapAt10Pct > 0 {
			Expect(ms.HeapAlloc).To(BeNumerically("<=", 2*heapAt10Pct), "heap did not plateau: 10%%-mark=%d final=%d (final > 2x 10%%-mark)", heapAt10Pct, ms.HeapAlloc)
		}
		if runtime.GOOS == "linux" && baselineFDs >= 0 {
			Expect(finalFDs).To(BeNumerically("<=", baselineFDs), "fd count grew: baseline=%d final=%d", baselineFDs, finalFDs)
		}
	})
})

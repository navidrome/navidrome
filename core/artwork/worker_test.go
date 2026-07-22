package artwork

import (
	"context"
	"errors"
	"io"
	"os"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/goleak"
)

// reenqueueOnDequeue simulates a concurrent scan Enqueue between DequeueBatch and the
// worker's delete by bumping retry_at, so a DeleteIfUnchanged on the dequeued value no-ops.
type reenqueueOnDequeue struct {
	*tests.MockArtworkQueueRepo
	done bool
}

func (r *reenqueueOnDequeue) DequeueBatch(n int) ([]model.ArtworkQueueItem, error) {
	items, err := r.MockArtworkQueueRepo.DequeueBatch(n)
	if !r.done && len(items) > 0 {
		r.done = true
		for k, it := range r.Data {
			if it.ItemKind == items[0].ItemKind && it.ItemID == items[0].ItemID {
				it.RetryAt = items[0].RetryAt.Add(time.Minute)
				r.Data[k] = it
			}
		}
	}
	return items, err
}

func findQueued(q *tests.MockArtworkQueueRepo, kind, id string) *model.ArtworkQueueItem {
	for _, it := range q.Data {
		if it.ItemKind == kind && it.ItemID == id {
			return &it
		}
	}
	return nil
}

var _ = Describe("Worker", func() {
	var (
		ctx        context.Context
		ds         *tests.MockDataStore
		folderRepo *fakeFolderRepo
		libRepo    *tests.MockLibraryRepo
		ffm        *tests.MockFFmpeg
		ag         *agents.Agents
		store      *ImageStore
		artRepo    *tests.MockArtworkRepo
		queueRepo  *tests.MockArtworkQueueRepo
		repoRoot   string
		w          *Worker
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = context.Background()
		var err error
		repoRoot, err = os.Getwd()
		Expect(err).ToNot(HaveOccurred())

		folderRepo = &fakeFolderRepo{}
		libRepo = &tests.MockLibraryRepo{}
		libRepo.SetData(model.Libraries{{ID: 0, Path: testFileLibPath(repoRoot)}})
		ffm = tests.NewMockFFmpeg("")
		ag = agents.GetAgents(&tests.MockDataStore{}, nil)
		artRepo = tests.CreateMockArtworkRepo()
		queueRepo = tests.CreateMockArtworkQueueRepo()
		ds = &tests.MockDataStore{
			MockedFolder:       folderRepo,
			MockedLibrary:      libRepo,
			MockedArtwork:      artRepo,
			MockedArtworkQueue: queueRepo,
		}
		ds.MockedAlbum = tests.CreateMockAlbumRepo()
		store = NewImageStore(GinkgoT().TempDir())
		conf.Server.CoverArtPriority = "cover.jpg, embedded"
		conf.Server.ArtworkExternalMaxRPS = 1000 // keep the limiter out of the way of behavior tests
		w = NewWorker(ds, store, ag, ffm)
	})

	Describe("drain", func() {
		It("processes a seeded queue item and removes it from the queue", func() {
			folderRepo.result = []model.Folder{{
				Path:       "tests/fixtures/artist/an-album",
				ImageFiles: []string{"cover.jpg"},
			}}
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
				{ID: "al1", Name: "Album", FolderIDs: []string{"f1"}},
			})
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{
				ItemKind: "al", ItemID: "al1", Priority: model.ArtworkPriorityScan,
			})).To(Succeed())

			n, err := w.drain(ctx, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			ia, err := artRepo.GetItemArtwork("al", "al1", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Source).To(Equal("folder"))

			count, err := queueRepo.Count()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(BeZero(), "a found item must be deleted from the queue")
		})

		It("reschedules a failed item via MarkFailed with a backed-off retry_at", func() {
			conf.Server.CoverArtPriority = "external"
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "al4", Name: "Album"}})
			imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{ItemKind: "al", ItemID: "al4"})).To(Succeed())

			n, err := w.drain(ctx, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			it := findQueued(queueRepo, "al", "al4")
			Expect(it).ToNot(BeNil())
			Expect(it.Attempts).To(Equal(1))
			Expect(it.RetryAt).To(BeTemporally(">", time.Now()))

			_, err = artRepo.GetItemArtwork("al", "al4", model.ImageTypePrimary)
			Expect(err).To(MatchError(model.ErrNotFound), "a timeout must never settle on absent")
		})

		It("reschedules a found-stale item via MarkFailed while keeping its served state", func() {
			conf.Server.CoverArtPriority = "external, cover.jpg"
			folderRepo.result = []model.Folder{{
				Path:       "tests/fixtures/artist/an-album",
				ImageFiles: []string{"cover.jpg"},
			}}
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
				{ID: "alstale", Name: "Album", FolderIDs: []string{"f1"}},
			})
			imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{ItemKind: "al", ItemID: "alstale"})).To(Succeed())

			n, err := w.drain(ctx, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			it := findQueued(queueRepo, "al", "alstale")
			Expect(it).ToNot(BeNil(), "a found-stale row must survive for a higher-priority retry")
			Expect(it.Attempts).To(Equal(1))
			Expect(it.RetryAt).To(BeTemporally(">", time.Now()))

			ia, err := artRepo.GetItemArtwork("al", "alstale", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Source).To(Equal("folder"), "the fallback art is served meanwhile")
		})

		It("keeps a row re-enqueued between dequeue and delete", func() {
			folderRepo.result = []model.Folder{{
				Path:       "tests/fixtures/artist/an-album",
				ImageFiles: []string{"cover.jpg"},
			}}
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
				{ID: "al7", Name: "Album", FolderIDs: []string{"f1"}},
			})
			racing := &reenqueueOnDequeue{MockArtworkQueueRepo: queueRepo}
			ds.MockedArtworkQueue = racing
			w = NewWorker(ds, store, ag, ffm)
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{
				ItemKind: "al", ItemID: "al7", Priority: model.ArtworkPriorityScan,
			})).To(Succeed())

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			// The concurrent re-enqueue changed retry_at, so the found-path delete was a no-op.
			Expect(findQueued(queueRepo, "al", "al7")).ToNot(BeNil())
			ia, err := artRepo.GetItemArtwork("al", "al7", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Source).To(Equal("folder"))
		})

		It("keeps a fresh re-enqueue ahead of a stale failure backoff", func() {
			conf.Server.CoverArtPriority = "external"
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "al8", Name: "Album"}})
			imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})
			racing := &reenqueueOnDequeue{MockArtworkQueueRepo: queueRepo}
			ds.MockedArtworkQueue = racing
			w = NewWorker(ds, store, ag, ffm)
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{ItemKind: "al", ItemID: "al8"})).To(Succeed())
			dequeued := findQueued(queueRepo, "al", "al8").RetryAt

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			// The concurrent re-enqueue reset retry_at; the failure path must not stomp it
			// with stale backoff nor bump attempts, so the row stays immediately eligible.
			it := findQueued(queueRepo, "al", "al8")
			Expect(it).ToNot(BeNil())
			Expect(it.Attempts).To(BeZero())
			Expect(it.RetryAt).To(BeTemporally("==", dequeued.Add(time.Minute)))
		})

		It("resolves a private playlist under an admin context instead of failing forever", func() {
			ds.MockedUser = adminUserRepo()
			vds := &visibilityPlaylistDS{
				MockDataStore: ds,
				private:       model.Playlist{ID: "plPriv", OwnerID: "admin"},
				tracks:        &tests.MockPlaylistTrackRepo{},
			}
			w = NewWorker(vds, store, ag, ffm)
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{ItemKind: "pl", ItemID: "plPriv"})).To(Succeed())

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			// Resolved as absent (no art) and removed — not stuck failing on ErrNotFound forever.
			Expect(findQueued(queueRepo, "pl", "plPriv")).To(BeNil())
			ia, err := artRepo.GetItemArtwork("pl", "plPriv", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Hash).To(BeEmpty())
		})

		It("returns zero when the queue is empty", func() {
			n, err := w.drain(ctx, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(BeZero())
		})
	})

	Describe("Bump", func() {
		It("enqueues at Bump priority and wakes the loop", func() {
			w.Bump("al", "al9")
			it := findQueued(queueRepo, "al", "al9")
			Expect(it).ToNot(BeNil())
			Expect(it.Priority).To(Equal(model.ArtworkPriorityBump))
		})
	})

	Describe("gate/breaker", func() {
		It("opens after 5 consecutive external errors and short-circuits the step", func() {
			var calls int
			failing := func() (io.ReadCloser, string, error) {
				calls++
				return nil, "", errors.New("boom")
			}
			for range 5 {
				_, _, err := w.gate("A", failing)
				Expect(err).To(HaveOccurred())
			}
			Expect(calls).To(Equal(5))

			_, _, err := w.gate("A", failing)
			Expect(err).To(MatchError(errBreakerOpen))
			Expect(calls).To(Equal(5), "an open breaker must not call the external step")
		})

		It("resets the failure count on a successful call", func() {
			failing := func() (io.ReadCloser, string, error) { return nil, "", errors.New("boom") }
			ok := func() (io.ReadCloser, string, error) { return io.NopCloser(nil), "p", nil }
			for range 4 {
				_, _, _ = w.gate("A", failing)
			}
			_, _, err := w.gate("A", ok)
			Expect(err).ToNot(HaveOccurred())

			var calls int
			counting := func() (io.ReadCloser, string, error) {
				calls++
				return nil, "", errors.New("boom")
			}
			for range 5 {
				_, _, _ = w.gate("A", counting)
			}
			Expect(calls).To(Equal(5), "the breaker should have re-closed after the success")
		})

		It("does not open the breaker on a run of agent not-found misses", func() {
			// Regression: agents.ErrNotFound is a definitive miss, not a fault. A run of
			// artless items must never trip the breaker, or they'd loop in retry instead of
			// settling absent. Uses the real gate, not passthroughGate.
			notFound := func() (io.ReadCloser, string, error) { return nil, "", agents.ErrNotFound }
			for range breakerThreshold + 3 {
				_, _, err := w.gate("A", notFound)
				Expect(err).To(MatchError(agents.ErrNotFound), "a miss passes through, never errBreakerOpen")
			}

			var calls int
			counting := func() (io.ReadCloser, string, error) {
				calls++
				return nil, "", errors.New("boom")
			}
			_, _, _ = w.gate("A", counting)
			Expect(calls).To(Equal(1), "the breaker stayed closed, so the step still runs")
		})

		It("isolates each agent's breaker: one open gate does not block another", func() {
			failing := func() (io.ReadCloser, string, error) { return nil, "", errors.New("boom") }
			for range breakerThreshold {
				_, _, _ = w.gate("A", failing)
			}
			_, _, err := w.gate("A", failing)
			Expect(err).To(MatchError(errBreakerOpen), "agent A's breaker is open")

			var bCalls int
			bStep := func() (io.ReadCloser, string, error) {
				bCalls++
				return io.NopCloser(nil), "p", nil
			}
			for range breakerThreshold + 2 {
				_, _, err := w.gate("B", bStep)
				Expect(err).ToNot(HaveOccurred(), "agent B keeps being called while A is open")
			}
			Expect(bCalls).To(Equal(breakerThreshold + 2))
		})
	})

	Describe("RunPrune", func() {
		It("runs a prune under the worker mutex", func() {
			Expect(w.RunPrune(ctx)).To(Succeed())
		})
	})

	Describe("Run", func() {
		It("exits cleanly when the context is cancelled", func() {
			runCtx, cancel := context.WithCancel(ctx)
			done := make(chan error, 1)
			go func() { done <- w.Run(runCtx) }()

			cancel()
			Eventually(done, time.Second).Should(Receive(BeNil()))
		})

		It("does not leak goroutines after Run exits", func() {
			DeferCleanup(configtest.SetupConfig())

			ignore := goleak.IgnoreCurrent()
			DeferCleanup(func() { goleak.VerifyNone(GinkgoT(), ignore) })

			localDS := &tests.MockDataStore{MockedArtworkQueue: tests.CreateMockArtworkQueueRepo()}
			lw := NewWorker(localDS, NewImageStore(GinkgoT().TempDir()), agents.GetAgents(localDS, nil), tests.NewMockFFmpeg(""))

			runCtx, cancel := context.WithCancel(ctx)
			done := make(chan error, 1)
			go func() { done <- lw.Run(runCtx) }()

			time.Sleep(20 * time.Millisecond) // let the loop settle on the idle select
			cancel()
			Eventually(done, 2*time.Second).Should(Receive(BeNil()))
		})
	})
})

var _ = Describe("backoff", func() {
	It("returns the expected schedule with no jitter", func() {
		for _, c := range []struct {
			attempts int
			want     time.Duration
		}{
			{0, 5 * time.Minute},
			{1, 20 * time.Minute},
			{2, 80 * time.Minute},
			{3, 320 * time.Minute},
			{4, 1280 * time.Minute},
			{5, 48 * time.Hour},
			{6, 48 * time.Hour},
		} {
			Expect(backoffFor(c.attempts, 0)).To(Equal(c.want), "attempt %d", c.attempts)
		}
	})

	It("applies jitter proportionally", func() {
		base := backoffFor(2, 0)
		Expect(backoffFor(2, 0.2)).To(Equal(time.Duration(float64(base) * 1.2)))
		Expect(backoffFor(2, -0.2)).To(Equal(time.Duration(float64(base) * 0.8)))
	})

	It("keeps random jitter within +/-20%", func() {
		lo := time.Duration(float64(320*time.Minute) * 0.8)
		hi := time.Duration(float64(320*time.Minute) * 1.2)
		for range 200 {
			d := backoff(3)
			Expect(d).To(BeNumerically(">=", lo))
			Expect(d).To(BeNumerically("<=", hi))
		}
	})
})

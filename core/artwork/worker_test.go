package artwork

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/cache"
	"github.com/navidrome/navidrome/utils/slice"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/goleak"
)

type recordingCache struct {
	cache.FileCache
	mu       sync.Mutex
	keys     []string
	disabled bool
}

func (c *recordingCache) Disabled(ctx context.Context) bool {
	return c.disabled || c.FileCache.Disabled(ctx)
}

func (c *recordingCache) Get(ctx context.Context, arg cache.Item) (*cache.CachedStream, error) {
	c.mu.Lock()
	c.keys = append(c.keys, arg.Key())
	c.mu.Unlock()
	return c.FileCache.Get(ctx, arg)
}

func (c *recordingCache) getKeys() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.keys)
}

// Simulates a concurrent Enqueue between DequeueBatch and the worker's delete, so
// DeleteIfUnchanged on the dequeued value no-ops.
type reenqueueOnDequeue struct {
	*tests.MockArtworkQueueRepo
	done bool
}

func (r *reenqueueOnDequeue) DequeueBatch(n int, kinds ...string) ([]model.ArtworkQueueItem, error) {
	items, err := r.MockArtworkQueueRepo.DequeueBatch(n, kinds...)
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

type fakeEventBroker struct {
	http.Handler
	mu     sync.Mutex
	events []events.Event
}

func (f *fakeEventBroker) SendMessage(_ context.Context, event events.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
}

func (f *fakeEventBroker) SendBroadcastMessage(_ context.Context, event events.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
}

func (f *fakeEventBroker) getEvents() []events.Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.events)
}

var _ events.Broker = (*fakeEventBroker)(nil)

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
		broker     *fakeEventBroker
		imgCache   *recordingCache
		repoRoot   string
		w          *Worker
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = context.Background()
		var err error
		repoRoot, err = os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		conf.Server.CacheFolder = conf.NewDir(GinkgoT().TempDir())

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
		conf.Server.DevArtworkExternalMaxRPS = 1000 // keep the limiter out of the way of behavior tests
		broker = &fakeEventBroker{}
		imgCache = &recordingCache{FileCache: cache.NewFileCache("WorkerTest", "100MB", "images", 0,
			func(ctx context.Context, arg cache.Item) (io.Reader, error) {
				return arg.(artworkReader).Reader(ctx)
			})}
		// Init walks the cache dir on a goroutine; loaded CI runners can take >1s.
		Eventually(func() bool { return imgCache.Available(ctx) }, 10*time.Second).Should(BeTrue())
		w = NewWorker(ds, store, ag, ffm, broker, imgCache)
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

			ia, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al1", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Source).To(Equal("folder"))

			count, err := queueRepo.Count()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(BeZero(), "a found item must be deleted from the queue")
		})

		It("processes an mf queue item, writing state and storing embedded bytes", func() {
			conf.Server.EnableMediaFileCoverArt = true
			ds.MockedMediaFile = tests.CreateMockMediaFileRepo()
			ds.MockedMediaFile.(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "mf1", LibraryID: 0, Path: "tests/fixtures/artist/an-album/test.mp3", HasCoverArt: true},
			})
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{
				ItemKind: "mf", ItemID: "mf1", Priority: model.ArtworkPriorityBump,
			})).To(Succeed())

			n, err := w.drain(ctx, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			ia, err := artRepo.GetItemArtwork(model.KindMediaFileArtwork, "mf1", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Source).To(Equal("embedded"))
			Expect(ia.Hash).ToNot(BeEmpty())

			art, err := artRepo.GetImage(ia.Hash)
			Expect(err).ToNot(HaveOccurred())
			r, err := store.Open(ia.Hash, art.Mime)
			Expect(err).ToNot(HaveOccurred())
			defer r.Close()
			data, err := io.ReadAll(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(data).ToNot(BeEmpty(), "embedded bytes must be written to the store")

			count, err := queueRepo.Count()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(BeZero())
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

			_, err = artRepo.GetItemArtwork(model.KindAlbumArtwork, "al4", model.ImageTypePrimary)
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

			ia, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "alstale", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Source).To(Equal("folder"), "the fallback art is served meanwhile")

			evts := broker.getEvents()
			Expect(evts).To(HaveLen(1), "the served fallback art must live-refresh the UI")
			Expect(evts[0].(*events.RefreshResource).Data(evts[0])).To(ContainSubstring("alstale"))
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
			w = NewWorker(ds, store, ag, ffm, broker, imgCache)
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{
				ItemKind: "al", ItemID: "al7", Priority: model.ArtworkPriorityScan,
			})).To(Succeed())

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			// The concurrent re-enqueue changed retry_at, so the found-path delete was a no-op.
			Expect(findQueued(queueRepo, "al", "al7")).ToNot(BeNil())
			ia, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al7", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Source).To(Equal("folder"))
		})

		It("keeps a fresh re-enqueue ahead of a stale failure backoff", func() {
			conf.Server.CoverArtPriority = "external"
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "al8", Name: "Album"}})
			imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})
			racing := &reenqueueOnDequeue{MockArtworkQueueRepo: queueRepo}
			ds.MockedArtworkQueue = racing
			w = NewWorker(ds, store, ag, ffm, broker, imgCache)
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{ItemKind: "al", ItemID: "al8"})).To(Succeed())
			dequeued := findQueued(queueRepo, "al", "al8").RetryAt

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			// The re-enqueue reset retry_at; the failure path must not stomp it nor bump attempts.
			it := findQueued(queueRepo, "al", "al8")
			Expect(it).ToNot(BeNil())
			Expect(it.Attempts).To(BeZero())
			Expect(it.RetryAt).To(BeTemporally("==", dequeued.Add(time.Minute)))
		})

		It("gives up and settles absent once the retry budget is exhausted", func() {
			conf.Server.CoverArtPriority = "external"
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "al9", Name: "Album"}})
			imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})
			w = NewWorker(ds, store, ag, ffm, broker, imgCache)
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{ItemKind: "al", ItemID: "al9"})).To(Succeed())
			// Age the row past the retry budget.
			for k, v := range queueRepo.Data {
				if v.ItemID == "al9" {
					v.EnqueuedAt = time.Now().Add(-(giveUpAfter + time.Hour))
					queueRepo.Data[k] = v
				}
			}

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			Expect(findQueued(queueRepo, "al", "al9")).To(BeNil())
			ia, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al9", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Hash).To(BeEmpty())
		})

		It("keeps already-served art when the retry budget is exhausted", func() {
			conf.Server.CoverArtPriority = "external"
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "al10", Name: "Album"}})
			Expect(artRepo.PutItemArtwork(&model.ItemArtwork{
				ItemKind: "al", ItemID: "al10", ImageType: model.ImageTypePrimary,
				Hash: "cafebabe", Source: "external:lastfm",
			})).To(Succeed())
			imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})
			w = NewWorker(ds, store, ag, ffm, broker, imgCache)
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{ItemKind: "al", ItemID: "al10"})).To(Succeed())
			for k, v := range queueRepo.Data {
				if v.ItemID == "al10" {
					v.EnqueuedAt = time.Now().Add(-(giveUpAfter + time.Hour))
					queueRepo.Data[k] = v
				}
			}

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			Expect(findQueued(queueRepo, "al", "al10")).To(BeNil())
			ia, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al10", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Hash).To(Equal("cafebabe"), "a persistent outage must not discard served art")
		})

		// Media files are excluded from recheckKinds, so an absent row here would never be
		// revisited: a transient read error would look permanent.
		It("does not settle absent on exhaustion for a kind with no recheck path", func() {
			conf.Server.EnableMediaFileCoverArt = true
			ds.MockedMediaFile = tests.CreateMockMediaFileRepo()
			ds.MockedMediaFile.(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "mfX", LibraryID: 0, Path: "tests/fixtures/artist/an-album/gone.mp3", HasCoverArt: true},
			})
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{ItemKind: "mf", ItemID: "mfX"})).To(Succeed())
			for k, v := range queueRepo.Data {
				if v.ItemID == "mfX" {
					v.EnqueuedAt = time.Now().Add(-(giveUpAfter + time.Hour))
					queueRepo.Data[k] = v
				}
			}

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			Expect(findQueued(queueRepo, "mf", "mfX")).To(BeNil(), "the row must stop retrying")
			_, err = artRepo.GetItemArtwork(model.KindMediaFileArtwork, "mfX", model.ImageTypePrimary)
			Expect(err).To(MatchError(model.ErrNotFound),
				"no row leaves the track unresolved, so a later view can still recover it")
		})

		It("resolves a private playlist under an admin context instead of failing forever", func() {
			ds.MockedUser = adminUserRepo()
			vds := &visibilityPlaylistDS{
				MockDataStore: ds,
				private:       model.Playlist{ID: "plPriv", OwnerID: "admin"},
				tracks:        &tests.MockPlaylistTrackRepo{},
			}
			w = NewWorker(vds, store, ag, ffm, broker, imgCache)
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{ItemKind: "pl", ItemID: "plPriv"})).To(Succeed())

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			Expect(findQueued(queueRepo, "pl", "plPriv")).To(BeNil())
			ia, err := artRepo.GetItemArtwork(model.KindPlaylistArtwork, "plPriv", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Hash).To(BeEmpty())
		})

		It("returns zero when the queue is empty", func() {
			n, err := w.drain(ctx, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(BeZero())
		})

		It("broadcasts a single refresh event for the found items in a batch", func() {
			folderRepo.result = []model.Folder{{
				Path:       "tests/fixtures/artist/an-album",
				ImageFiles: []string{"cover.jpg"},
			}}
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
				{ID: "al1", Name: "Album 1", FolderIDs: []string{"f1"}},
				{ID: "al2", Name: "Album 2", FolderIDs: []string{"f1"}},
			})
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{ItemKind: "al", ItemID: "al1", Priority: model.ArtworkPriorityScan})).To(Succeed())
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{ItemKind: "al", ItemID: "al2", Priority: model.ArtworkPriorityScan})).To(Succeed())
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar1", Priority: model.ArtworkPriorityScan})).To(Succeed())

			n, err := w.drain(ctx, 3)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(3))

			evts := broker.getEvents()
			Expect(evts).To(HaveLen(1), "exactly one coalesced event per drain batch")
			rr, ok := evts[0].(*events.RefreshResource)
			Expect(ok).To(BeTrue())
			data := rr.Data(rr)
			Expect(data).To(ContainSubstring(`"album"`))
			Expect(data).To(ContainSubstring("al1"))
			Expect(data).To(ContainSubstring("al2"))
			Expect(data).ToNot(ContainSubstring("artist"), "a failed (unresolved) artist must not be refreshed")
			Expect(data).ToNot(ContainSubstring("ar1"))
		})

		DescribeTable("only lists the kinds it actually resolved",
			func(kinds []string, wantSong bool) {
				items := slice.Map(kinds, func(k string) model.ArtworkQueueItem {
					return model.ArtworkQueueItem{ItemKind: k, ItemID: k + "1"}
				})
				w.broadcastRefresh(ctx, items)

				evts := broker.getEvents()
				Expect(evts).To(HaveLen(1))
				data := evts[0].(*events.RefreshResource).Data(evts[0])
				if wantSong {
					Expect(data).To(ContainSubstring(`"song"`))
				} else {
					Expect(data).ToNot(ContainSubstring(`"song"`))
				}
			},
			// An album's tracks inherit its art, but the dependent ids are unbounded: the client
			// fans an album refresh out to the tracks it has loaded.
			Entry("album alone does not name songs", []string{"al"}, false),
			Entry("artist alone does not", []string{"ar"}, false),
			Entry("playlist alone does not", []string{"pl"}, false),
			Entry("album mixed with others still does not", []string{"ar", "al"}, false),
			Entry("songs resolving on their own are listed by id", []string{"mf"}, true),
		)

		It("broadcasts a refresh when an item resolves to absent (removed cover)", func() {
			conf.Server.CoverArtPriority = "cover.*" // local-only; no folder image → absent
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "al3", Name: "Artless"}})
			folderRepo.result = nil
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{ItemKind: "al", ItemID: "al3", Priority: model.ArtworkPriorityScan})).To(Succeed())

			n, err := w.drain(ctx, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			evts := broker.getEvents()
			Expect(evts).To(HaveLen(1), "a removed cover must live-refresh clients so they drop it")
			Expect(evts[0].(*events.RefreshResource).Data(evts[0])).To(ContainSubstring("al3"))

			ia, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al3", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Hash).To(BeEmpty(), "the outcome was absent, not found")
		})

		It("does not broadcast when no item is found", func() {
			conf.Server.CoverArtPriority = "external"
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "alx", Name: "Album"}})
			imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{ItemKind: "al", ItemID: "alx"})).To(Succeed())

			n, err := w.drain(ctx, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			Expect(broker.getEvents()).To(BeEmpty(), "a drain with no found items sends no event")
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
			// agents.ErrNotFound is a definitive miss, not a fault: artless items must not
			// trip the breaker, or they would loop in retry instead of settling absent.
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

	Describe("precache", func() {
		BeforeEach(func() {
			folderRepo.result = []model.Folder{{
				Path:       "tests/fixtures/artist/an-album",
				ImageFiles: []string{"cover.jpg"},
			}}
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
				{ID: "alpc", Name: "Album", FolderIDs: []string{"f1"}},
			})
			conf.Server.UICoverArtSize = 300
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{
				ItemKind: "al", ItemID: "alpc", Priority: model.ArtworkPriorityScan,
			})).To(Succeed())
		})

		It("warms the resize cache at the UI cover size after a found acquisition", func() {
			conf.Server.EnableArtworkPrecache = true

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			// The list surfaces request square covers, so warming any other key is wasted work.
			Expect(imgCache.getKeys()).To(ContainElement(ContainSubstring(".300.true.")))
		})

		It("skips warming when precache is disabled", func() {
			conf.Server.EnableArtworkPrecache = false

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			Expect(imgCache.getKeys()).To(BeEmpty())
		})

		// It warms from the bytes acquisition already held, so no state row or store file is needed.
		It("warms from the acquired bytes without reading them back", func() {
			conf.Server.EnableArtworkPrecache = true
			ia := &model.ItemArtwork{
				ItemKind: "al", ItemID: "unpersisted", ImageType: model.ImageTypePrimary,
				Hash: "abcdef0123456789", UpdatedAt: time.Now(),
			}

			data, err := os.ReadFile("tests/fixtures/artist/an-album/cover.jpg")
			Expect(err).ToNot(HaveOccurred())

			w.precache(ctx, &acquired{ia: ia, mime: "image/jpeg", data: data})

			// Nothing backs that hash on disk, so a hit can only come from the bytes handed in;
			// the probe refuses to open, proving nothing is re-read.
			probe := &resizedItem{
				hash: ia.Hash, size: 300, square: true, ffmpeg: ffm,
				open: func() (io.ReadCloser, error) { return nil, errors.New("precache must not re-read the source") },
			}
			stream, err := imgCache.Get(ctx, probe)
			Expect(err).ToNot(HaveOccurred())
			defer stream.Close()
			Expect(io.ReadAll(stream)).ToNot(BeEmpty())
		})
	})

	Describe("drain pools", func() {
		// A kind in neither pool is never dequeued, with nothing to catch it at compile time.
		It("covers every kind the worker can process, exactly once", func() {
			var pooled []string
			for _, p := range newDrainPools() {
				pooled = append(pooled, p.kinds...)
			}
			for kind := range artworkKindToResource {
				Expect(pooled).To(ContainElement(kind.Prefix()), "kind %q belongs to no drain pool", kind.Prefix())
			}
			Expect(pooled).To(HaveLen(len(artworkKindToResource)), "a kind is claimed by more than one pool")
		})

		// Artists resolve through a rate-limited agent that holds its slot while waiting, so one
		// shared pool would park every album behind them.
		It("resolves albums while artists are stuck on a slow agent", func() {
			conf.Server.CoverArtPriority = "cover.jpg"
			conf.Server.ArtistArtPriority = "external"
			folderRepo.result = []model.Folder{{
				Path:       "tests/fixtures/artist/an-album",
				ImageFiles: []string{"cover.jpg"},
			}}
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "alx", Name: "Album", FolderIDs: []string{"f1"}}})
			ds.MockedArtist = tests.CreateMockArtistRepo()

			// Every artist lookup blocks until released, standing in for the rate limiter.
			block := make(chan struct{})
			artists := model.Artists{}
			for i := range 8 {
				artists = append(artists, model.Artist{ID: fmt.Sprintf("arx%d", i), Name: "A"})
			}
			ds.MockedArtist.(*tests.MockArtistRepo).SetData(artists)
			imageAgents(&fakeImageAgent{name: "slowAgent", block: block})

			// Artists first, exactly as Backfill orders them.
			for _, a := range artists {
				Expect(queueRepo.Enqueue(model.ArtworkQueueItem{
					ItemKind: "ar", ItemID: a.ID, Priority: model.ArtworkPriorityBackfill,
				})).To(Succeed())
			}
			Expect(queueRepo.Enqueue(model.ArtworkQueueItem{
				ItemKind: "al", ItemID: "alx", Priority: model.ArtworkPriorityBackfill,
			})).To(Succeed())

			runCtx, cancel := context.WithCancel(ctx)
			done := make(chan struct{})
			go func() { defer close(done); _ = w.Run(runCtx) }()
			// Join Run: a leaked pool goroutine would race the config snapshot Ginkgo restores.
			DeferCleanup(func() {
				cancel()
				close(block) // unpark the blocked lookups so the pools can unwind
				<-done
			})

			Eventually(func() bool {
				ia, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "alx", model.ImageTypePrimary)
				return err == nil && ia.Hash != ""
			}, 5*time.Second, 50*time.Millisecond).Should(BeTrue(),
				"a blocked external pool must not hold up local artwork")

			_, err := artRepo.GetItemArtwork(model.KindArtistArtwork, "arx0", model.ImageTypePrimary)
			Expect(err).To(MatchError(model.ErrNotFound), "artists are still blocked, as intended")
		})
	})

	Describe("batching", func() {
		It("leaves undispatched items queued when cancelled mid-batch", func() {
			// Every album must resolve (to absent), so ANY dispatched item deletes its row
			// and the assertion below can see it — regardless of dequeue order.
			albums := model.Albums{}
			for i := range 8 {
				id := fmt.Sprintf("alc%d", i)
				albums = append(albums, model.Album{ID: id, Name: "Album"})
				Expect(queueRepo.Enqueue(model.ArtworkQueueItem{
					ItemKind: "al", ItemID: id, Priority: model.ArtworkPriorityScan,
				})).To(Succeed())
			}
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(albums)
			cancelledCtx, cancel := context.WithCancel(ctx)
			cancel()

			_, err := w.drain(cancelledCtx, 1)
			Expect(err).ToNot(HaveOccurred())

			for i := range 8 {
				id := fmt.Sprintf("alc%d", i)
				Expect(findQueued(queueRepo, "al", id)).ToNot(BeNil(), "row "+id+" must survive a cancelled drain")
			}
		})

		It("dequeues past the worker pool so one drain covers many items", func() {
			for i := range 16 {
				ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{{ID: fmt.Sprintf("alb%d", i), Name: "Album"}})
				Expect(queueRepo.Enqueue(model.ArtworkQueueItem{
					ItemKind: "al", ItemID: fmt.Sprintf("alb%d", i), Priority: model.ArtworkPriorityScan,
				})).To(Succeed())
			}

			n, err := w.drain(ctx, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(16), "a batch sized to the pool would have stopped at 4")
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
			lw := NewWorker(localDS, NewImageStore(GinkgoT().TempDir()), agents.GetAgents(localDS, nil), tests.NewMockFFmpeg(""), &fakeEventBroker{}, imgCache)

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
			{0, 5 * time.Second},
			{1, 20 * time.Second},
			{2, 80 * time.Second},
			{3, 320 * time.Second},
			{4, 1280 * time.Second},
			{5, 5120 * time.Second},
			{6, 20480 * time.Second},
			{7, 12 * time.Hour},
			{8, 12 * time.Hour},
		} {
			Expect(backoffFor(c.attempts, 0)).To(Equal(c.want), "attempt %d", c.attempts)
		}
	})

	It("applies jitter proportionally", func() {
		base := backoffFor(2, 0)
		Expect(backoffFor(2, 0.2)).To(Equal(time.Duration(float64(base) * 1.2)))
		Expect(backoffFor(2, -0.2)).To(Equal(time.Duration(float64(base) * 0.8)))
	})

	It("keeps random jitter within +/-40%", func() {
		lo := time.Duration(float64(320*time.Second) * 0.6)
		hi := time.Duration(float64(320*time.Second) * 1.4)
		for range 200 {
			d := backoff(3)
			Expect(d).To(BeNumerically(">=", lo))
			Expect(d).To(BeNumerically("<=", hi))
		}
	})
})

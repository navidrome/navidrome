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
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence/persistencetest"
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
	model.ArtworkQueueRepository
	done bool
}

func (r *reenqueueOnDequeue) DequeueBatch(ctx context.Context, n int, kinds ...string) ([]model.ArtworkQueueItem, error) {
	items, err := r.ArtworkQueueRepository.DequeueBatch(ctx, n, kinds...)
	if err != nil || r.done || len(items) == 0 {
		return items, err
	}
	r.done = true
	_, err = db.Db().ExecContext(ctx, "UPDATE artwork_queue SET retry_at = ? WHERE item_kind = ? AND item_id = ?",
		items[0].RetryAt.Add(time.Minute), items[0].ItemKind, items[0].ItemID)
	return items, err
}

func withQueue(ds model.DataStore, queue model.ArtworkQueueRepository) model.DataStore {
	return persistencetest.Stub(ds, persistencetest.Repos{
		ArtworkQueue: func(model.ArtworkQueueRepository) model.ArtworkQueueRepository { return queue },
	})
}

func withRacingQueue(ds model.DataStore) model.DataStore {
	return withQueue(ds, &reenqueueOnDequeue{ArtworkQueueRepository: ds.ArtworkQueue()})
}

// cancelOnDequeue cancels the drain right after its batch is read, so nothing is dispatched.
type cancelOnDequeue struct {
	model.ArtworkQueueRepository
	cancel context.CancelFunc
}

func (c *cancelOnDequeue) DequeueBatch(ctx context.Context, n int, kinds ...string) ([]model.ArtworkQueueItem, error) {
	defer c.cancel()
	return c.ArtworkQueueRepository.DequeueBatch(ctx, n, kinds...)
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

type panickingAlbumRepo struct {
	model.AlbumRepository
	panicID string
}

func (r *panickingAlbumRepo) Get(ctx context.Context, id string) (*model.Album, error) {
	if id == r.panicID {
		panic("boom") //nolint:forbidigo
	}
	return r.AlbumRepository.Get(ctx, id)
}

var _ = Describe("Worker", func() {
	var (
		ctx      context.Context
		ds       model.DataStore
		ffm      *tests.MockFFmpeg
		ag       *agents.Agents
		store    *ImageStore
		broker   *fakeEventBroker
		imgCache *recordingCache
		w        *Worker
	)

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		repoRoot, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		conf.Update(func(c *conf.Options) { c.CacheFolder = conf.NewDir(GinkgoT().TempDir()) })

		ds = newTestDS(ctx)
		useLibrary(ctx, ds, repoRoot)
		ffm = tests.NewMockFFmpeg("")
		ag = agents.GetAgents(ds, nil)
		store = NewImageStore(GinkgoT().TempDir())
		conf.Update(func(c *conf.Options) {
			c.CoverArtPriority = "cover.jpg, embedded"
			c.DevArtworkExternalMaxRPS = 1000
		}) // keep the limiter out of the way of behavior tests
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
			fid := putFolder(ctx, ds, fixtureAlbumDir, "cover.jpg")
			persistencetest.PutAlbums(ctx, ds, model.Album{ID: "al1", Name: "Album", FolderIDs: []string{fid}})
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{
				ItemKind: "al", ItemID: "al1", Priority: model.ArtworkPriorityScan,
			})).To(Succeed())

			n, err := w.drain(ctx, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			ia, err := ds.Artwork().GetItemArtwork(ctx, model.KindAlbumArtwork, "al1", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Source).To(Equal("folder"))

			Expect(queueCount(ctx, ds)).To(BeZero(), "a found item must be deleted from the queue")
		})

		It("processes an mf queue item, writing state and storing embedded bytes", func() {
			conf.Update(func(c *conf.Options) { c.EnableMediaFileCoverArt = true })
			persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "mf1", Path: fixtureAlbumDir + "/test.mp3", HasCoverArt: true})
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{
				ItemKind: "mf", ItemID: "mf1", Priority: model.ArtworkPriorityBump,
			})).To(Succeed())

			n, err := w.drain(ctx, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			ia, err := ds.Artwork().GetItemArtwork(ctx, model.KindMediaFileArtwork, "mf1", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Source).To(Equal("embedded"))
			Expect(ia.Hash).ToNot(BeEmpty())

			art, err := ds.Artwork().GetImage(ctx, ia.Hash)
			Expect(err).ToNot(HaveOccurred())
			r, err := store.Open(ia.Hash, art.Mime)
			Expect(err).ToNot(HaveOccurred())
			defer r.Close()
			data, err := io.ReadAll(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(data).ToNot(BeEmpty(), "embedded bytes must be written to the store")

			Expect(queueCount(ctx, ds)).To(BeZero())
		})

		It("reschedules a failed item via MarkFailed with a backed-off retry_at", func() {
			conf.Update(func(c *conf.Options) { c.CoverArtPriority = "external" })
			persistencetest.PutAlbums(ctx, ds, model.Album{ID: "al4", Name: "Album"})
			imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al4"})).To(Succeed())

			n, err := w.drain(ctx, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			it := queued(ctx, ds, "al", "al4")
			Expect(it).ToNot(BeNil())
			Expect(it.Attempts).To(Equal(1))
			Expect(it.RetryAt).To(BeTemporally(">", time.Now()))

			_, err = ds.Artwork().GetItemArtwork(ctx, model.KindAlbumArtwork, "al4", model.ImageTypePrimary)
			Expect(err).To(MatchError(model.ErrNotFound), "a timeout must never settle on absent")
		})

		It("fails an item that panics, without stopping the rest of the batch", func() {
			fid := putFolder(ctx, ds, fixtureAlbumDir, "cover.jpg")
			persistencetest.PutAlbums(ctx, ds,
				model.Album{ID: "alboom", Name: "Album", FolderIDs: []string{fid}},
				model.Album{ID: "alok", Name: "Album", FolderIDs: []string{fid}},
			)
			panicky := persistencetest.Stub(ds, persistencetest.Repos{Album: func(r model.AlbumRepository) model.AlbumRepository {
				return &panickingAlbumRepo{AlbumRepository: r, panicID: "alboom"}
			}})
			w = NewWorker(panicky, store, ag, ffm, broker, imgCache)
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "alboom"})).To(Succeed())
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "alok"})).To(Succeed())

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(2))

			it := queued(ctx, ds, "al", "alboom")
			Expect(it).ToNot(BeNil(), "a panicking item must be rescheduled, not dropped")
			Expect(it.Attempts).To(Equal(1))
			Expect(it.RetryAt).To(BeTemporally(">", time.Now()))
			Expect(it.Trace).To(ContainSubstring("boom"))

			Expect(queued(ctx, ds, "al", "alok")).To(BeNil())
			ia, err := ds.Artwork().GetItemArtwork(ctx, model.KindAlbumArtwork, "alok", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Source).To(Equal("folder"))
		})

		It("reschedules past the provider's requested delay when it exceeds the backoff", func() {
			conf.Update(func(c *conf.Options) { c.CoverArtPriority = "external" })
			persistencetest.PutAlbums(ctx, ds, model.Album{ID: "al9", Name: "Album"})
			// Well above backoff(0)'s jittered ceiling, so only the hint can produce this retry_at.
			const askedFor = 90 * time.Minute
			imageAgents(&fakeImageAgent{name: "throttledAgent", err: &agents.RetryLaterError{RetryIn: askedFor}})
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al9"})).To(Succeed())

			n, err := w.drain(ctx, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			it := queued(ctx, ds, "al", "al9")
			Expect(it).ToNot(BeNil())
			Expect(it.RetryAt).To(BeTemporally("~", time.Now().Add(askedFor), time.Minute))
		})

		It("reschedules a found-stale item via MarkFailed while keeping its served state", func() {
			conf.Update(func(c *conf.Options) { c.CoverArtPriority = "external, cover.jpg" })
			fid := putFolder(ctx, ds, fixtureAlbumDir, "cover.jpg")
			persistencetest.PutAlbums(ctx, ds, model.Album{ID: "alstale", Name: "Album", FolderIDs: []string{fid}})
			imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "alstale"})).To(Succeed())

			n, err := w.drain(ctx, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			it := queued(ctx, ds, "al", "alstale")
			Expect(it).ToNot(BeNil(), "a found-stale row must survive for a higher-priority retry")
			Expect(it.Attempts).To(Equal(1))
			Expect(it.RetryAt).To(BeTemporally(">", time.Now()))

			ia, err := ds.Artwork().GetItemArtwork(ctx, model.KindAlbumArtwork, "alstale", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Source).To(Equal("folder"), "the fallback art is served meanwhile")

			evts := broker.getEvents()
			Expect(evts).To(HaveLen(1), "the served fallback art must live-refresh the UI")
			Expect(evts[0].(*events.RefreshResource).Data(evts[0])).To(ContainSubstring("alstale"))
		})

		It("keeps a row re-enqueued between dequeue and delete", func() {
			fid := putFolder(ctx, ds, fixtureAlbumDir, "cover.jpg")
			persistencetest.PutAlbums(ctx, ds, model.Album{ID: "al7", Name: "Album", FolderIDs: []string{fid}})
			w = NewWorker(withRacingQueue(ds), store, ag, ffm, broker, imgCache)
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{
				ItemKind: "al", ItemID: "al7", Priority: model.ArtworkPriorityScan,
			})).To(Succeed())

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			// The concurrent re-enqueue changed retry_at, so the found-path delete was a no-op.
			Expect(queued(ctx, ds, "al", "al7")).ToNot(BeNil())
			ia, err := ds.Artwork().GetItemArtwork(ctx, model.KindAlbumArtwork, "al7", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Source).To(Equal("folder"))
		})

		It("keeps a fresh re-enqueue ahead of a stale failure backoff", func() {
			conf.Update(func(c *conf.Options) { c.CoverArtPriority = "external" })
			persistencetest.PutAlbums(ctx, ds, model.Album{ID: "al8", Name: "Album"})
			imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})
			w = NewWorker(withRacingQueue(ds), store, ag, ffm, broker, imgCache)
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al8"})).To(Succeed())
			dequeued := queued(ctx, ds, "al", "al8").RetryAt

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			// The re-enqueue reset retry_at; the failure path must not stomp it nor bump attempts.
			it := queued(ctx, ds, "al", "al8")
			Expect(it).ToNot(BeNil())
			Expect(it.Attempts).To(BeZero())
			Expect(it.RetryAt).To(BeTemporally("==", dequeued.Add(time.Minute)))
		})

		It("gives up and settles absent once the retry budget is exhausted", func() {
			conf.Update(func(c *conf.Options) { c.CoverArtPriority = "external" })
			persistencetest.PutAlbums(ctx, ds, model.Album{ID: "al9", Name: "Album"})
			imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})
			w = NewWorker(ds, store, ag, ffm, broker, imgCache)
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al9"})).To(Succeed())
			// Age the row past the retry budget.
			expireQueued("al", "al9")

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			Expect(queued(ctx, ds, "al", "al9")).To(BeNil())
			ia, err := ds.Artwork().GetItemArtwork(ctx, model.KindAlbumArtwork, "al9", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Hash).To(BeEmpty())
		})

		It("keeps already-served art when the retry budget is exhausted", func() {
			conf.Update(func(c *conf.Options) { c.CoverArtPriority = "external" })
			persistencetest.PutAlbums(ctx, ds, model.Album{ID: "al10", Name: "Album"})
			Expect(ds.Artwork().PutItemArtwork(ctx, &model.ItemArtwork{
				ItemKind: "al", ItemID: "al10", ImageType: model.ImageTypePrimary,
				Hash: "cafebabe", Source: "external:lastfm",
			})).To(Succeed())
			imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})
			w = NewWorker(ds, store, ag, ffm, broker, imgCache)
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al10"})).To(Succeed())
			expireQueued("al", "al10")

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			Expect(queued(ctx, ds, "al", "al10")).To(BeNil())
			ia, err := ds.Artwork().GetItemArtwork(ctx, model.KindAlbumArtwork, "al10", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Hash).To(Equal("cafebabe"), "a persistent outage must not discard served art")
		})

		It("records on the queue row why the last attempt failed", func() {
			conf.Update(func(c *conf.Options) { c.CoverArtPriority = "external" })
			persistencetest.PutAlbums(ctx, ds, model.Album{ID: "al11", Name: "Album"})
			imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})
			w = NewWorker(ds, store, ag, ffm, broker, imgCache)
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al11"})).To(Succeed())

			_, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())

			it := queued(ctx, ds, "al", "al11")
			Expect(it).ToNot(BeNil())
			Expect(DecodeTrace(it.Trace, "")).To(ContainElement(SatisfyAll(
				HaveField("Candidate", "external:failAgent"),
				HaveField("Outcome", OutcomeError),
				HaveField("Detail", ContainSubstring("agent timed out")),
			)), "a retrying row must say why it is retrying")
		})

		// The give-up path settles absent before recording, so the row exists by the time the
		// failure is written. Recording first would silently lose it for every unresolved item.
		It("keeps the failure for an item that never resolved at all", func() {
			conf.Update(func(c *conf.Options) { c.CoverArtPriority = "external" })
			persistencetest.PutAlbums(ctx, ds, model.Album{ID: "al13", Name: "Album"})
			imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})
			w = NewWorker(ds, store, ag, ffm, broker, imgCache)
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al13"})).To(Succeed())
			expireQueued("al", "al13")

			_, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())

			ia, err := ds.Artwork().GetItemArtwork(ctx, model.KindAlbumArtwork, "al13", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred(), "settling absent must create the row the failure is written to")
			Expect(ia.Hash).To(BeEmpty())
			Expect(DecodeTrace(ia.LastFailure, "")).ToNot(BeEmpty())
		})

		It("keeps the failure on the state row after the queue row is deleted", func() {
			conf.Update(func(c *conf.Options) { c.CoverArtPriority = "external" })
			persistencetest.PutAlbums(ctx, ds, model.Album{ID: "al12", Name: "Album"})
			Expect(ds.Artwork().PutItemArtwork(ctx, &model.ItemArtwork{
				ItemKind: "al", ItemID: "al12", ImageType: model.ImageTypePrimary,
				Hash: "cafebabe", Source: "external:lastfm",
			})).To(Succeed())
			imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})
			w = NewWorker(ds, store, ag, ffm, broker, imgCache)
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al12"})).To(Succeed())
			expireQueued("al", "al12")

			_, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())

			Expect(queued(ctx, ds, "al", "al12")).To(BeNil())
			ia, err := ds.Artwork().GetItemArtwork(ctx, model.KindAlbumArtwork, "al12", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(DecodeTrace(ia.LastFailure, "")).ToNot(BeEmpty(),
				"the queue row is gone, so this is the only remaining record of the failure")
			Expect(ia.Hash).To(Equal("cafebabe"), "recording the failure must not disturb the served art")
		})

		// Only a view enqueues a media file, and an absent row is exactly what stops a view from
		// doing so: a transient read error would look permanent.
		It("does not settle absent on exhaustion for a media file", func() {
			conf.Update(func(c *conf.Options) { c.EnableMediaFileCoverArt = true })
			persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "mfX", Path: fixtureAlbumDir + "/gone.mp3", HasCoverArt: true})
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{ItemKind: "mf", ItemID: "mfX"})).To(Succeed())
			expireQueued("mf", "mfX")

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			Expect(queued(ctx, ds, "mf", "mfX")).To(BeNil(), "the row must stop retrying")
			_, err = ds.Artwork().GetItemArtwork(ctx, model.KindMediaFileArtwork, "mfX", model.ImageTypePrimary)
			Expect(err).To(MatchError(model.ErrNotFound),
				"no row leaves the track unresolved, so a later view can still recover it")
			// Known gap: with no row and no absent settle, there is nowhere to keep the failure.
			// Creating one here would write an empty hash, which every reader treats as absent.
		})

		It("resolves a private playlist under an admin context instead of failing forever", func() {
			putPlaylist(ctx, ds, model.Playlist{ID: "plPriv", Name: "Private", Public: false})
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{ItemKind: "pl", ItemID: "plPriv"})).To(Succeed())

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			Expect(queued(ctx, ds, "pl", "plPriv")).To(BeNil())
			ia, err := ds.Artwork().GetItemArtwork(ctx, model.KindPlaylistArtwork, "plPriv", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Hash).To(BeEmpty())
		})

		It("returns zero when the queue is empty", func() {
			n, err := w.drain(ctx, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(BeZero())
		})

		It("broadcasts a single refresh event for the found items in a batch", func() {
			fid := putFolder(ctx, ds, fixtureAlbumDir, "cover.jpg")
			persistencetest.PutAlbums(ctx, ds,
				model.Album{ID: "al1", Name: "Album 1", FolderIDs: []string{fid}},
				model.Album{ID: "al2", Name: "Album 2", FolderIDs: []string{fid}},
			)
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al1", Priority: model.ArtworkPriorityScan})).To(Succeed())
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al2", Priority: model.ArtworkPriorityScan})).To(Succeed())
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar1", Priority: model.ArtworkPriorityScan})).To(Succeed())

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
			conf.Update(func(c *conf.Options) { c.CoverArtPriority = "cover.*" }) // local-only; no folder image → absent
			persistencetest.PutAlbums(ctx, ds, model.Album{ID: "al3", Name: "Artless"})
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al3", Priority: model.ArtworkPriorityScan})).To(Succeed())

			n, err := w.drain(ctx, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			evts := broker.getEvents()
			Expect(evts).To(HaveLen(1), "a removed cover must live-refresh clients so they drop it")
			Expect(evts[0].(*events.RefreshResource).Data(evts[0])).To(ContainSubstring("al3"))

			ia, err := ds.Artwork().GetItemArtwork(ctx, model.KindAlbumArtwork, "al3", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Hash).To(BeEmpty(), "the outcome was absent, not found")
		})

		It("does not broadcast when no item is found", func() {
			conf.Update(func(c *conf.Options) { c.CoverArtPriority = "external" })
			persistencetest.PutAlbums(ctx, ds, model.Album{ID: "alx", Name: "Album"})
			imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "alx"})).To(Succeed())

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

		It("does not open the breaker when the run is cancelled", func() {
			cancelled := func() (io.ReadCloser, string, error) { return nil, "", context.Canceled }
			for range breakerThreshold + 3 {
				_, _, err := w.gate("A", cancelled)
				Expect(err).To(MatchError(context.Canceled), "a cancellation passes through, never errBreakerOpen")
			}

			var calls int
			counting := func() (io.ReadCloser, string, error) {
				calls++
				return nil, "", errors.New("boom")
			}
			_, _, _ = w.gate("A", counting)
			Expect(calls).To(Equal(1), "the breaker stayed closed, so the step still runs")
		})

		It("ignores a cancellation mid-run, neither counting nor clearing the failures", func() {
			failing := func() (io.ReadCloser, string, error) { return nil, "", errors.New("boom") }
			cancelled := func() (io.ReadCloser, string, error) { return nil, "", context.Canceled }
			for range breakerThreshold - 1 {
				_, _, _ = w.gate("A", failing)
			}
			_, _, _ = w.gate("A", cancelled)

			var calls int
			counting := func() (io.ReadCloser, string, error) {
				calls++
				return nil, "", errors.New("boom")
			}
			_, _, _ = w.gate("A", counting)
			Expect(calls).To(Equal(1), "the cancellation must not have counted as the final failure")

			_, _, err := w.gate("A", counting)
			Expect(err).To(MatchError(errBreakerOpen), "the cancellation must not have cleared the earlier failures")
			Expect(calls).To(Equal(1), "an open breaker must not call the external step")
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
			fid := putFolder(ctx, ds, fixtureAlbumDir, "cover.jpg")
			persistencetest.PutAlbums(ctx, ds, model.Album{ID: "alpc", Name: "Album", FolderIDs: []string{fid}})
			conf.Update(func(c *conf.Options) { c.UICoverArtSize = 300 })
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{
				ItemKind: "al", ItemID: "alpc", Priority: model.ArtworkPriorityScan,
			})).To(Succeed())
		})

		It("warms the resize cache at the UI cover size after a found acquisition", func() {
			conf.Update(func(c *conf.Options) { c.EnableArtworkPrecache = true })

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			// The list surfaces request square covers, so warming any other key is wasted work.
			Expect(imgCache.getKeys()).To(ContainElement(ContainSubstring(".300.true.")))
		})

		It("skips warming when precache is disabled", func() {
			conf.Update(func(c *conf.Options) { c.EnableArtworkPrecache = false })

			n, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(1))

			Expect(imgCache.getKeys()).To(BeEmpty())
		})

		// It warms from the bytes acquisition already held, so no state row or store file is needed.
		It("warms from the acquired bytes without reading them back", func() {
			conf.Update(func(c *conf.Options) { c.EnableArtworkPrecache = true })
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
			conf.Update(func(c *conf.Options) {
				c.CoverArtPriority = "cover.jpg"
				c.ArtistArtPriority = "external"
			})
			fid := putFolder(ctx, ds, fixtureAlbumDir, "cover.jpg")
			persistencetest.PutAlbums(ctx, ds, model.Album{ID: "alx", Name: "Album", FolderIDs: []string{fid}})

			// Every artist lookup blocks until released, standing in for the rate limiter.
			block := make(chan struct{})
			artists := model.Artists{}
			for i := range 8 {
				artists = append(artists, model.Artist{ID: fmt.Sprintf("arx%d", i), Name: "A"})
			}
			persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artists...)
			imageAgents(&fakeImageAgent{name: "slowAgent", block: block})

			// Artists first, exactly as Backfill orders them.
			for _, a := range artists {
				Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{
					ItemKind: "ar", ItemID: a.ID, Priority: model.ArtworkPriorityBackfill,
				})).To(Succeed())
			}
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{
				ItemKind: "al", ItemID: "alx", Priority: model.ArtworkPriorityBackfill,
			})).To(Succeed())

			runCtx, cancel := context.WithCancel(ctx)
			done := make(chan struct{})
			go func() { defer close(done); _ = w.Run(runCtx) }()
			// Join Run so no pool goroutine outlives the spec.
			DeferCleanup(func() {
				cancel()
				close(block) // unpark the blocked lookups so the pools can unwind
				<-done
			})

			Eventually(func() bool {
				ia, err := ds.Artwork().GetItemArtwork(ctx, model.KindAlbumArtwork, "alx", model.ImageTypePrimary)
				return err == nil && ia.Hash != ""
			}, 5*time.Second, 50*time.Millisecond).Should(BeTrue(),
				"a blocked external pool must not hold up local artwork")

			_, err := ds.Artwork().GetItemArtwork(ctx, model.KindArtistArtwork, "arx0", model.ImageTypePrimary)
			Expect(err).To(MatchError(model.ErrNotFound), "artists are still blocked, as intended")
		})
	})

	Describe("batching", func() {
		It("leaves undispatched items queued when cancelled mid-batch", func() {
			// Every album must resolve, so any dispatched item deletes its row regardless of dequeue order.
			albums := model.Albums{}
			for i := range 8 {
				id := fmt.Sprintf("alc%d", i)
				albums = append(albums, model.Album{ID: id, Name: "Album"})
				Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{
					ItemKind: "al", ItemID: id, Priority: model.ArtworkPriorityScan,
				})).To(Succeed())
			}
			persistencetest.PutAlbums(ctx, ds, albums...)
			drainCtx, cancel := context.WithCancel(ctx)
			w = NewWorker(withQueue(ds, &cancelOnDequeue{ArtworkQueueRepository: ds.ArtworkQueue(), cancel: cancel}),
				store, ag, ffm, broker, imgCache)

			n, err := w.drain(drainCtx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(8), "the whole batch was read before the cancellation")

			for i := range 8 {
				id := fmt.Sprintf("alc%d", i)
				Expect(queued(ctx, ds, "al", id)).ToNot(BeNil(), "row "+id+" must survive a cancelled drain")
			}
		})

		It("stops dispatching and leaves the rest queued when paused mid-batch", func() {
			fid := putFolder(ctx, ds, fixtureAlbumDir, "cover.jpg")
			albums := model.Albums{}
			for i := range 8 {
				id := fmt.Sprintf("alp%d", i)
				albums = append(albums, model.Album{ID: id, Name: "Album", FolderIDs: []string{fid}})
				Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{
					ItemKind: "al", ItemID: id, Priority: model.ArtworkPriorityScan,
				})).To(Succeed())
			}
			persistencetest.PutAlbums(ctx, ds, albums...)
			// Pauses as soon as the first item has left the queue.
			w.PauseWhile(func() bool {
				n, _ := ds.ArtworkQueue().Count(ctx)
				return n < 8
			})

			_, err := w.drain(ctx, 1)
			Expect(err).ToNot(HaveOccurred())

			Expect(queueCount(ctx, ds)).To(Equal(int64(7)), "only the item dispatched before the pause may leave the queue")
		})

		It("dequeues past the worker pool so one drain covers many items", func() {
			for i := range 16 {
				persistencetest.PutAlbums(ctx, ds, model.Album{ID: fmt.Sprintf("alb%d", i), Name: "Album"})
				Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{
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

		It("does not drain the queue while paused", func() {
			fid := putFolder(ctx, ds, fixtureAlbumDir, "cover.jpg")
			persistencetest.PutAlbums(ctx, ds, model.Album{ID: "al1", Name: "Album", FolderIDs: []string{fid}})
			Expect(ds.ArtworkQueue().Enqueue(ctx, model.ArtworkQueueItem{
				ItemKind: "al", ItemID: "al1", Priority: model.ArtworkPriorityScan,
			})).To(Succeed())
			w.PauseWhile(func() bool { return true })

			runCtx, cancel := context.WithCancel(ctx)
			done := make(chan error, 1)
			go func() { done <- w.Run(runCtx) }()
			DeferCleanup(func() {
				cancel()
				Eventually(done, 2*time.Second).Should(Receive(BeNil()))
			})

			Consistently(func() any { return queued(ctx, ds, "al", "al1") }, 300*time.Millisecond).ShouldNot(BeNil())
		})

		It("does not leak goroutines after Run exits", func() {

			ignore := goleak.IgnoreCurrent()
			DeferCleanup(func() { goleak.VerifyNone(GinkgoT(), ignore) })

			lw := NewWorker(ds, NewImageStore(GinkgoT().TempDir()), ag, tests.NewMockFFmpeg(""), &fakeEventBroker{}, imgCache)

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

var _ = Describe("retryDelay", func() {
	It("uses the backoff schedule when the provider asked for nothing", func() {
		d := retryDelay(0, 0)
		Expect(d).To(BeNumerically(">=", 3*time.Second))
		Expect(d).To(BeNumerically("<=", 7*time.Second))
	})

	It("waits the provider's delay when it is longer than the backoff", func() {
		Expect(retryDelay(0, time.Hour)).To(Equal(time.Hour))
	})

	It("keeps the backoff when it is longer than the provider's delay", func() {
		Expect(retryDelay(4, time.Second)).To(BeNumerically(">=", 3*time.Second))
	})
})

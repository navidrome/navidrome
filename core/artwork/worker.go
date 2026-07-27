package artwork

import (
	"bytes"
	"context"
	"io"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/utils/cache"
)

const (
	workerPollInterval = 5 * time.Second
	backoffBase        = 5 * time.Second
	// giveUpAfter bounds the retry budget from enqueue: past it the worker stops retrying and
	// hands the item to the periodic stale-absent recheck (settling absent on a bare failure).
	giveUpAfter = 12 * time.Hour
)

// drainPool drains one class of work with its own slot budget, so a kind whose resolution
// blocks cannot occupy slots another kind needs.
type drainPool struct {
	name        string
	kinds       []string
	concurrency int
	wake        chan struct{}
}

// Worker drains the artwork queue through the processor: each external agent is rate-limited
// and circuit-broken independently, and prune is serialized against the store-write window
// via pruneMu.
type Worker struct {
	proc    *processor
	cache   cache.FileCache
	ffmpeg  ffmpeg.FFmpeg
	broker  events.Broker
	pruneMu sync.RWMutex
	pools   []*drainPool
	runCtx  context.Context

	gatesMu sync.Mutex
	gates   map[string]*extGate
}

func NewWorker(ds model.DataStore, store *ImageStore, ag *agents.Agents, ffmpeg ffmpeg.FFmpeg, broker events.Broker, imgCache cache.FileCache) *Worker {
	w := &Worker{
		proc:   &processor{ds: ds, store: store},
		cache:  imgCache,
		ffmpeg: ffmpeg,
		broker: broker,
		pools:  newDrainPools(),
		runCtx: context.Background(),
		gates:  map[string]*extGate{},
	}
	w.proc.resolver = newResolver(ds, ag, ffmpeg, w.gate)
	w.proc.pruneLock = w.pruneMu.RLocker()
	return w
}

// newDrainPools splits the drain by what bounds it: gate() waits for its rate-limit permit
// while holding a slot, so a sleeping lookup would otherwise crowd out a cover sitting on disk.
func newDrainPools() []*drainPool {
	budget := conf.MaxOpenConns() // floored at 4, so both remainders below stay positive
	local := min(max(1, conf.Server.ArtworkWorkerConcurrency), budget-1)
	// More external slots than the rate allows would only sleep in the limiter.
	external := min(max(2, 2*conf.Server.ArtworkExternalMaxRPS), budget-local)
	return []*drainPool{
		{name: "local", kinds: localDrainKinds, concurrency: local, wake: make(chan struct{}, 1)},
		{name: "external", kinds: externalDrainKinds, concurrency: external, wake: make(chan struct{}, 1)},
	}
}

// Kind is a proxy for cost: an album whose chain reaches "external" still costs a local slot,
// but only the few with no local art do.
var (
	externalDrainKinds = []string{model.KindArtistArtwork.Prefix()}
	localDrainKinds    = []string{
		model.KindAlbumArtwork.Prefix(),
		model.KindPlaylistArtwork.Prefix(),
		model.KindRadioArtwork.Prefix(),
		model.KindMediaFileArtwork.Prefix(),
	}
)

// Run blocks draining the queue until ctx is cancelled. It exits cleanly with no
// leaked goroutines: each drain waits for its batch before the loop can return.
func (w *Worker) Run(ctx context.Context) error {
	w.runCtx = ctx
	var wg sync.WaitGroup
	for _, p := range w.pools {
		wg.Go(func() { w.runPool(ctx, p) })
	}
	wg.Wait()
	return nil
}

// runPool drains one pool's kinds until ctx is cancelled.
func (w *Worker) runPool(ctx context.Context, p *drainPool) {
	ticker := time.NewTicker(workerPollInterval)
	defer ticker.Stop()
	for {
		n, err := w.drain(ctx, p.concurrency, p.kinds...)
		if err != nil && ctx.Err() == nil {
			log.Warn(ctx, "Artwork: Worker drain failed", "pool", p.name, err)
		}
		if ctx.Err() != nil {
			return
		}
		if n > 0 {
			continue // keep draining while this pool has ready work
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-p.wake:
		}
	}
}

// Bump enqueues an item at the highest priority and wakes the drain loop. It is
// non-blocking: a wake already pending is enough.
func (w *Worker) Bump(kind, id string) {
	item := model.ArtworkQueueItem{
		ItemKind:  kind,
		ItemID:    id,
		ImageType: model.ImageTypePrimary,
		Priority:  model.ArtworkPriorityBump,
	}
	if err := w.proc.ds.ArtworkQueue(context.Background()).Enqueue(item); err != nil {
		log.Warn("Artwork: Could not bump queue item", "kind", kind, "id", id, err)
		return
	}
	// Waking all beats routing by kind: a spurious wake costs one empty dequeue, while an
	// unrouted kind would never wake at all.
	for _, p := range w.pools {
		select {
		case p.wake <- struct{}{}:
		default:
		}
	}
}

// RunPrune runs prune under the worker's write lock, so no acquisition can place
// a file while orphans are being reclaimed. This is the only sanctioned prune path.
func (w *Worker) RunPrune(ctx context.Context) error {
	w.pruneMu.Lock()
	defer w.pruneMu.Unlock()
	return prune(ctx, w.proc.ds, w.proc.store)
}

// Backfill enqueues every entity for re-resolution when the artwork config fingerprint has
// changed since the last run, artists first. It reports whether anything was enqueued.
func (w *Worker) Backfill(ctx context.Context) (bool, error) {
	return backfill(ctx, w.proc.ds)
}

// EnqueueStaleAbsentAll requeues known-absent entries older than staleAbsentAge, so artwork
// that appeared since the last attempt is eventually picked up.
func (w *Worker) EnqueueStaleAbsentAll(ctx context.Context) error {
	return enqueueStaleAbsentAll(ctx, w.proc.ds)
}

// EnqueueMissingAll requeues entities that have no artwork state row at all: the safety net
// for anything a scan never enqueued.
func (w *Worker) EnqueueMissingAll(ctx context.Context) error {
	return enqueueMissingAll(ctx, w.proc.ds)
}

func (w *Worker) drain(ctx context.Context, concurrency int, kinds ...string) (int, error) {
	// Dequeue well past the worker pool so a slow item (an external lookup burning its
	// timeout) never idles the other slots: the pool stays fed until the batch runs out.
	// DequeueBatch does not mark rows taken, so this is one query per pass, not per slot.
	items, err := w.proc.ds.ArtworkQueue(ctx).DequeueBatch(max(16, 4*concurrency), kinds...)
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, nil
	}
	drainStart := time.Now()
	// Resolved only once there is work, and per drain rather than per item: the worker needs an
	// admin identity for private playlists, and can start before any admin exists.
	ctx = auth.WithAdminUser(ctx, w.proc.ds)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var refreshMu sync.Mutex
	var refresh []model.ArtworkQueueItem
	for _, item := range items {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			wg.Wait()
			return len(items), nil
		}
		wg.Go(func() {
			defer func() { <-sem }()
			out, got := w.process(ctx, item)
			// Refresh clients on any visible state change: found/foundStale (new art) and absent
			// (removed art — clients must drop a previously-served immutable cover). foundStale
			// also wrote a served state row.
			if out == outcomeFound || out == outcomeFoundStale || out == outcomeAbsent {
				refreshMu.Lock()
				refresh = append(refresh, item)
				refreshMu.Unlock()
			}
			// Precache only actual images. Post-outcome only: the queue row was already settled
			// by process, so warming the resize cache here can never block or alter queue ops.
			if got != nil {
				w.precache(ctx, got)
			}
		})
	}
	wg.Wait()
	w.broadcastRefresh(ctx, refresh)
	log.Debug(ctx, "Artwork: Drained a batch", "kinds", kinds, "items", len(items),
		"refreshed", len(refresh), "concurrency", concurrency, "elapsed", time.Since(drainStart))
	return len(items), nil
}

// artworkKindToResource maps an artwork kind to the UI resource name carried in the refresh
// event; note media_file maps to "song", so this can't derive from Kind.String().
var artworkKindToResource = map[model.Kind]string{
	model.KindAlbumArtwork:     "album",
	model.KindArtistArtwork:    "artist",
	model.KindPlaylistArtwork:  "playlist",
	model.KindRadioArtwork:     "radio",
	model.KindMediaFileArtwork: "song",
}

// broadcastRefresh emits one coalesced RefreshResource for the batch's newly-acquired
// artwork, so connected UIs re-fetch the affected records (and pick up the new coverArt id).
func (w *Worker) broadcastRefresh(ctx context.Context, found []model.ArtworkQueueItem) {
	if len(found) == 0 {
		return
	}
	event := &events.RefreshResource{}
	byResource := map[string][]string{}
	for _, it := range found {
		kind, _ := model.ParseKind(it.ItemKind)
		if res, ok := artworkKindToResource[kind]; ok {
			byResource[res] = append(byResource[res], it.ItemID)
		}
	}
	if len(byResource) == 0 {
		return
	}
	for res, ids := range byResource {
		event = event.With(res, ids...)
	}
	w.broker.SendBroadcastMessage(ctx, event)
}

func (w *Worker) process(ctx context.Context, item model.ArtworkQueueItem) (outcome, *acquired) {
	if item.ImageType == "" {
		item.ImageType = model.ImageTypePrimary
	}
	out, got := w.proc.acquire(ctx, item)

	queue := w.proc.ds.ArtworkQueue(ctx)
	switch out {
	case outcomeFound, outcomeAbsent:
		// DeleteIfUnchanged, not Delete: a scan that re-enqueued this row mid-flight reset
		// its retry_at, so the row survives here and the next drain re-resolves it.
		if err := queue.DeleteIfUnchanged(item.ItemKind, item.ItemID, item.ImageType, item.RetryAt); err != nil {
			log.Warn(ctx, "Artwork: Could not delete processed queue item", "kind", item.ItemKind, "id", item.ItemID, err)
		}
	case outcomeFoundStale, outcomeFailed:
		retryAt := time.Now().Add(backoff(item.Attempts))
		if retryAt.Before(item.EnqueuedAt.Add(giveUpAfter)) {
			// MarkFailedIfUnchanged, not MarkFailed: a scan that re-enqueued this row mid-flight reset
			// retry_at, so stale backoff must not stomp its fresh, immediate eligibility.
			if err := queue.MarkFailedIfUnchanged(item.ItemKind, item.ItemID, item.ImageType, item.RetryAt, retryAt); err != nil {
				log.Warn(ctx, "Artwork: Could not reschedule failed queue item", "kind", item.ItemKind, "id", item.ItemID, err)
			}
			log.Debug(ctx, "Artwork: Rescheduled item", "kind", item.ItemKind, "id", item.ItemID,
				"outcome", out, "attempts", item.Attempts+1, "retryIn", time.Until(retryAt),
				"budgetLeft", time.Until(item.EnqueuedAt.Add(giveUpAfter)))
			break
		}
		// Retry budget exhausted: stop retrying. Absent is only recoverable where a periodic
		// recheck will revisit it, so kinds without one keep no row at all; and art already
		// being served is kept, since exhaustion means the source stayed unreachable rather
		// than that the entity lost its cover.
		settled := "kept previous state"
		if out == outcomeFailed && hasRecheckPath(item.ItemKind) && !w.hasResolvedArtwork(ctx, item) {
			writeAbsent(ctx, w.proc.ds.Artwork(ctx), item)
			settled = "recorded absent"
		}
		log.Info(ctx, "Artwork: Retry budget exhausted, giving up", "kind", item.ItemKind, "id", item.ItemID,
			"outcome", out, "attempts", item.Attempts+1, "budget", giveUpAfter, "settled", settled)
		if err := queue.DeleteIfUnchanged(item.ItemKind, item.ItemID, item.ImageType, item.RetryAt); err != nil {
			log.Warn(ctx, "Artwork: Could not remove exhausted queue item", "kind", item.ItemKind, "id", item.ItemID, err)
		}
	}
	return out, got
}

// hasResolvedArtwork reports whether the item already has a hash-bearing state row.
func (w *Worker) hasResolvedArtwork(ctx context.Context, item model.ArtworkQueueItem) bool {
	kind, ok := model.ParseKind(item.ItemKind)
	if !ok {
		return false
	}
	ia, err := w.proc.ds.Artwork(ctx).GetItemArtwork(kind, item.ItemID, item.ImageType)
	return err == nil && ia.Hash != ""
}

// precache warms the resize cache at the UI cover size from the bytes just acquired, so the
// first UI request is a cache hit without re-reading the rows or the file. Skipped when
// disabled; failures are debug-only.
func (w *Worker) precache(ctx context.Context, got *acquired) {
	if !conf.Server.EnableArtworkPrecache || w.cache == nil || w.cache.Disabled(ctx) {
		return
	}
	precacheStart := time.Now()
	// Same key as the serving path (hash/size/square); only the source of the bytes differs.
	// square matches what the list surfaces request, otherwise this warms a key nothing reads.
	item := &resizedItem{
		hash:   got.ia.Hash,
		size:   conf.Server.UICoverArtSize,
		square: true,
		ffmpeg: w.ffmpeg,
		open:   func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(got.data)), nil },
	}
	stream, err := w.cache.Get(ctx, item)
	if err != nil {
		log.Debug(ctx, "Artwork: Precache failed", "kind", got.ia.ItemKind, "id", got.ia.ItemID, err)
		return
	}
	_, _ = io.Copy(io.Discard, stream)
	_ = stream.Close()
	log.Trace(ctx, "Artwork: Precached UI size", "kind", got.ia.ItemKind, "id", got.ia.ItemID,
		"size", conf.Server.UICoverArtSize, "elapsed", time.Since(precacheStart))
}

// backoffFor returns min(5s×4^n, giveUpAfter) scaled by (1+jitter), with jitter in [-0.4, 0.4].
func backoffFor(attempts int, jitter float64) time.Duration {
	d := math.Min(float64(backoffBase)*math.Pow(4, float64(attempts)), float64(giveUpAfter))
	return time.Duration(d * (1 + jitter))
}

func backoff(attempts int) time.Duration {
	return backoffFor(attempts, rand.Float64()*0.8-0.4) //nolint:gosec // retry jitter, not security-sensitive
}

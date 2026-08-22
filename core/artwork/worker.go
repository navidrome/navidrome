package artwork

import (
	"bytes"
	"cmp"
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
	// giveUpAfter bounds the retry budget from enqueue; past it the item falls to the
	// periodic stale-absent recheck.
	giveUpAfter = 12 * time.Hour
)

// drainPool drains one class of work with its own slot budget, so a blocking kind cannot
// occupy slots another kind needs.
type drainPool struct {
	name        string
	kinds       []string
	concurrency int
}

// Worker drains the artwork queue: each external agent is rate-limited and circuit-broken
// independently, and pruneMu serializes prune against the store-write window.
type Worker struct {
	proc    *processor
	agents  *agents.Agents
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
		agents: ag,
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

// newDrainPools splits the drain by what bounds it: gate() holds a slot while waiting for its
// rate-limit permit, so a sleeping lookup would crowd out a cover sitting on disk.
func newDrainPools() []*drainPool {
	budget := conf.MaxOpenConns() // floored at 4, so both remainders below stay positive
	local := min(max(1, conf.Server.DevArtworkWorkerConcurrency), budget-1)
	// More external slots than the rate allows would only sleep in the limiter.
	external := min(max(2, 2*conf.Server.DevArtworkExternalMaxRPS), budget-local)
	return []*drainPool{
		{name: "local", kinds: localDrainKinds, concurrency: local},
		{name: "external", kinds: externalDrainKinds, concurrency: external},
	}
}

// Kind is a proxy for cost: an album that reaches an external agent still costs a local slot.
var (
	externalDrainKinds = []string{model.KindArtistArtwork.Prefix()}
	localDrainKinds    = []string{
		model.KindAlbumArtwork.Prefix(),
		model.KindPlaylistArtwork.Prefix(),
		model.KindRadioArtwork.Prefix(),
		model.KindMediaFileArtwork.Prefix(),
	}
)

// Run blocks draining the queue until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	w.runCtx = ctx
	var wg sync.WaitGroup
	for _, p := range w.pools {
		wg.Go(func() { w.runPool(ctx, p) })
	}
	wg.Wait()
	return nil
}

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
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
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

// Backfill enqueues every entity for re-resolution when the artwork config fingerprint changed,
// artists first. It reports whether the backfill ran.
func (w *Worker) Backfill(ctx context.Context) (bool, error) {
	s, err := backfill(ctx, w.proc.ds, func() ImageAgentCount { return NewImageAgentCount(w.agents) })
	return s.Ran, err
}

// EnqueueStaleAbsentAll requeues known-absent entries older than StaleAbsentAge, at most
// StaleAbsentRecheckBatch per kind, oldest first.
func (w *Worker) EnqueueStaleAbsentAll(ctx context.Context) error {
	return enqueueStaleAbsentAll(ctx, w.proc.ds)
}

// EnqueueMissingAll requeues entities with no artwork state row: the safety net for anything
// a scan never enqueued.
func (w *Worker) EnqueueMissingAll(ctx context.Context) error {
	return enqueueMissingAll(ctx, w.proc.ds)
}

func (w *Worker) drain(ctx context.Context, concurrency int, kinds ...string) (int, error) {
	// Dequeue well past the pool size so a slow external lookup never idles the other slots.
	// DequeueBatch does not mark rows taken, so this is one query per pass, not per slot.
	items, err := w.proc.ds.ArtworkQueue(ctx).DequeueBatch(max(16, 4*concurrency), kinds...)
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, nil
	}
	drainStart := time.Now()
	// Private playlists need an admin identity; resolved per drain because the worker can
	// start before any admin exists.
	ctx = auth.WithAdminUser(ctx, w.proc.ds)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var refreshMu sync.Mutex
	var refresh []model.ArtworkQueueItem
	for _, item := range items {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
		}
		// select picks randomly when both cases are ready, so re-check to never dispatch after cancellation.
		if ctx.Err() != nil {
			wg.Wait()
			return len(items), nil //nolint:nilerr // a cancelled drain is a clean stop, not an error
		}
		wg.Go(func() {
			defer func() { <-sem }()
			out, got := w.process(ctx, item)
			// Absent counts as a visible change too: clients must drop a previously-served
			// immutable cover.
			if out == outcomeFound || out == outcomeFoundStale || out == outcomeAbsent {
				refreshMu.Lock()
				refresh = append(refresh, item)
				refreshMu.Unlock()
			}
			// Post-outcome: the queue row is already settled, so warming the cache can't
			// block or alter queue ops.
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

// artworkKindToResource maps a kind to its UI resource name; media_file maps to "song", so
// this can't derive from Kind.String().
var artworkKindToResource = map[model.Kind]string{
	model.KindAlbumArtwork:     "album",
	model.KindArtistArtwork:    "artist",
	model.KindPlaylistArtwork:  "playlist",
	model.KindRadioArtwork:     "radio",
	model.KindMediaFileArtwork: "song",
}

// broadcastRefresh emits one coalesced RefreshResource for the batch, so UIs re-fetch the
// affected records and pick up the new coverArt id.
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
	item.ImageType = cmp.Or(item.ImageType, model.ImageTypePrimary)
	trace := &ChainTrace{}
	ctx = withTrace(ctx, trace)
	out, got := w.proc.acquire(ctx, item)

	queue := w.proc.ds.ArtworkQueue(ctx)
	switch out {
	case outcomeFound, outcomeAbsent:
		// A scan that re-enqueued this row mid-flight reset its retry_at, so the row survives
		// here and the next drain re-resolves it.
		if err := queue.DeleteIfUnchanged(item.ItemKind, item.ItemID, item.ImageType, item.RetryAt); err != nil {
			log.Warn(ctx, "Artwork: Could not delete processed queue item", "kind", item.ItemKind, "id", item.ItemID, err)
		}
	case outcomeFoundStale, outcomeFailed:
		retryAt := time.Now().Add(backoff(item.Attempts))
		encoded := trace.encode("")
		if retryAt.Before(item.EnqueuedAt.Add(giveUpAfter)) {
			// A mid-flight re-enqueue reset retry_at; stale backoff must not stomp its
			// fresh, immediate eligibility.
			if err := queue.MarkFailedIfUnchanged(item.ItemKind, item.ItemID, item.ImageType, item.RetryAt, retryAt, encoded); err != nil {
				log.Warn(ctx, "Artwork: Could not reschedule failed queue item", "kind", item.ItemKind, "id", item.ItemID, err)
			}
			log.Debug(ctx, "Artwork: Rescheduled item", "kind", item.ItemKind, "id", item.ItemID,
				"outcome", out, "attempts", item.Attempts+1, "retryIn", time.Until(retryAt),
				"budgetLeft", time.Until(item.EnqueuedAt.Add(giveUpAfter)))
			break
		}
		// Absent is only recoverable where a periodic recheck revisits it, so other kinds keep
		// no row; art already being served is kept, as exhaustion means unreachable, not removed.
		settled := "kept previous state"
		if out == outcomeFailed && hasRecheckPath(item.ItemKind) && !w.hasResolvedArtwork(ctx, item) {
			writeAbsent(ctx, w.proc.ds.Artwork(ctx), item)
			settled = "recorded absent"
		}
		// The queue row is about to go, taking the only record of the failure with it. This write is
		// unconditional (not CAS-guarded) — safe only because the drain resolves each item serially.
		w.recordGiveUp(ctx, item, encoded)
		log.Info(ctx, "Artwork: Retry budget exhausted, giving up", "kind", item.ItemKind, "id", item.ItemID,
			"outcome", out, "attempts", item.Attempts+1, "budget", giveUpAfter, "settled", settled)
		if err := queue.DeleteIfUnchanged(item.ItemKind, item.ItemID, item.ImageType, item.RetryAt); err != nil {
			log.Warn(ctx, "Artwork: Could not remove exhausted queue item", "kind", item.ItemKind, "id", item.ItemID, err)
		}
	}
	return out, got
}

// recordGiveUp keeps the last failure on the state row after the queue row is deleted. An item
// that never resolved has no row to update, and creating one would settle it absent.
func (w *Worker) recordGiveUp(ctx context.Context, item model.ArtworkQueueItem, trace string) {
	kind, ok := model.ParseKind(item.ItemKind)
	if !ok {
		return
	}
	if err := w.proc.ds.Artwork(ctx).PutLastFailure(kind, item.ItemID, item.ImageType, trace); err != nil {
		log.Warn(ctx, "Artwork: Could not record the last failure", "kind", item.ItemKind, "id", item.ItemID, err)
	}
}

func (w *Worker) hasResolvedArtwork(ctx context.Context, item model.ArtworkQueueItem) bool {
	kind, ok := model.ParseKind(item.ItemKind)
	if !ok {
		return false
	}
	ia, err := w.proc.ds.Artwork(ctx).GetItemArtwork(kind, item.ItemID, item.ImageType)
	return err == nil && ia.Hash != ""
}

// precache warms the resize cache at the UI cover size from the bytes just acquired, so the
// first UI request hits without re-reading the rows or the file.
func (w *Worker) precache(ctx context.Context, got *acquired) {
	if !conf.Server.EnableArtworkPrecache || w.cache == nil || w.cache.Disabled(ctx) {
		return
	}
	precacheStart := time.Now()
	// Same key as the serving path: square must match what the list surfaces request, or this
	// warms a key nothing reads.
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
	d := min(float64(backoffBase)*math.Pow(4, float64(attempts)), float64(giveUpAfter))
	return time.Duration(d * (1 + jitter))
}

func backoff(attempts int) time.Duration {
	return backoffFor(attempts, rand.Float64()*0.8-0.4) //nolint:gosec // retry jitter, not security-sensitive
}

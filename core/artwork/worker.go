package artwork

import (
	"bytes"
	"context"
	"errors"
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
	"golang.org/x/time/rate"
)

const (
	workerPollInterval = 5 * time.Second
	backoffBase        = 5 * time.Second
	// giveUpAfter bounds the retry budget from enqueue: past it the worker stops retrying and
	// hands the item to the periodic stale-absent recheck (settling absent on a bare failure).
	giveUpAfter       = 12 * time.Hour
	breakerThreshold  = 5
	breakerProbeAfter = time.Minute
)

var errBreakerOpen = errors.New("artwork: external circuit breaker open")

// extGate is one agent's rate limiter + circuit breaker; each external agent gets its
// own so a provider whose API or CDN is down backs off in isolation from the others.
type extGate struct {
	limiter *rate.Limiter
	breaker *breaker
}

// drainPool drains one class of work with its own slot budget, so a kind whose resolution
// blocks cannot occupy slots another kind needs.
type drainPool struct {
	name        string
	kinds       []string
	concurrency int
	wake        chan struct{}
}

// Worker drains the artwork queue through processItem: each external agent is rate-limited
// and circuit-broken independently, and prune is serialized against in-flight acquisitions
// via pruneMu.
type Worker struct {
	deps    workerDeps
	broker  events.Broker
	pruneMu sync.RWMutex
	pools   []*drainPool
	runCtx  context.Context

	gatesMu sync.Mutex
	gates   map[string]*extGate

	mu       sync.Mutex
	inFlight map[string]struct{}
}

func NewWorker(ds model.DataStore, store *ImageStore, ag *agents.Agents, ffmpeg ffmpeg.FFmpeg, broker events.Broker, imgCache cache.FileCache) *Worker {
	w := &Worker{
		deps:     workerDeps{ds: ds, store: store, agents: ag, ffmpeg: ffmpeg, cache: imgCache},
		broker:   broker,
		pools:    newDrainPools(),
		runCtx:   context.Background(),
		gates:    map[string]*extGate{},
		inFlight: map[string]struct{}{},
	}
	w.deps.gate = w.gate
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
			log.Warn(ctx, "artwork: worker drain failed", "pool", p.name, err)
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
	if err := w.deps.ds.ArtworkQueue(context.Background()).Enqueue(item); err != nil {
		log.Warn("artwork: could not bump queue item", "kind", kind, "id", id, err)
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

// RunPrune runs Prune under the worker's write lock, so no acquisition can place
// a file while orphans are being reclaimed. This is the only sanctioned prune path.
func (w *Worker) RunPrune(ctx context.Context) error {
	w.pruneMu.Lock()
	defer w.pruneMu.Unlock()
	return Prune(ctx, w.deps.ds, w.deps.store)
}

func (w *Worker) drain(ctx context.Context, concurrency int, kinds ...string) (int, error) {
	// Dequeue well past the worker pool so a slow item (an external lookup burning its
	// timeout) never idles the other slots: the pool stays fed until the batch runs out.
	// DequeueBatch does not mark rows taken, so this is one query per pass, not per slot.
	batch, err := w.deps.ds.ArtworkQueue(ctx).DequeueBatch(max(16, 4*concurrency), kinds...)
	if err != nil {
		return 0, err
	}
	items := w.claim(batch)
	if len(items) == 0 {
		return 0, nil
	}
	// Resolved only once there is work, and per drain rather than per item: the worker needs an
	// admin identity for private playlists, and can start before any admin exists.
	ctx = auth.WithAdminUser(ctx, w.deps.ds)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var refreshMu sync.Mutex
	var refresh []model.ArtworkQueueItem
	for i, item := range items {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			// claim() reserved the whole batch; anything not dispatched has to go back, or it
			// stays in flight forever and no later drain can pick it up.
			for _, undispatched := range items[i:] {
				w.release(undispatched)
			}
			wg.Wait()
			return len(items), nil
		}
		wg.Add(1)
		go func(it model.ArtworkQueueItem) {
			defer wg.Done()
			defer func() { <-sem }()
			defer w.release(it)
			out, got := w.process(ctx, it)
			// Refresh clients on any visible state change: found/foundStale (new art) and absent
			// (removed art — clients must drop a previously-served immutable cover). foundStale
			// also wrote a served state row.
			if out == outcomeFound || out == outcomeFoundStale || out == outcomeAbsent {
				refreshMu.Lock()
				refresh = append(refresh, it)
				refreshMu.Unlock()
			}
			// Precache only actual images. Post-outcome only: the queue row was already settled
			// by process, so warming the resize cache here can never block or alter queue ops.
			if got != nil {
				w.precache(ctx, got)
			}
		}(item)
	}
	wg.Wait()
	w.broadcastRefresh(ctx, refresh)
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
	w.pruneMu.RLock()
	out, got := processItem(ctx, &w.deps, item)
	w.pruneMu.RUnlock()

	queue := w.deps.ds.ArtworkQueue(ctx)
	switch out {
	case outcomeFound, outcomeAbsent:
		// DeleteIfUnchanged, not Delete: a scan that re-enqueued this row mid-flight reset
		// its retry_at, so the row survives here and the next drain re-resolves it.
		if err := queue.DeleteIfUnchanged(item.ItemKind, item.ItemID, item.ImageType, item.RetryAt); err != nil {
			log.Warn(ctx, "artwork: could not delete processed queue item", "kind", item.ItemKind, "id", item.ItemID, err)
		}
	case outcomeFoundStale, outcomeFailed:
		retryAt := time.Now().Add(backoff(item.Attempts))
		if retryAt.Before(item.EnqueuedAt.Add(giveUpAfter)) {
			// MarkFailedIfUnchanged, not MarkFailed: a scan that re-enqueued this row mid-flight reset
			// retry_at, so stale backoff must not stomp its fresh, immediate eligibility.
			if err := queue.MarkFailedIfUnchanged(item.ItemKind, item.ItemID, item.ImageType, item.RetryAt, retryAt); err != nil {
				log.Warn(ctx, "artwork: could not reschedule failed queue item", "kind", item.ItemKind, "id", item.ItemID, err)
			}
			break
		}
		// Retry budget exhausted: stop retrying. A bare failure settles absent so the stale-absent
		// sweep (and a page view) can still recover it; a stale-found keeps its already-served art.
		// Art we are already serving is kept too: exhaustion means the source stayed unreachable,
		// not that the entity lost its cover.
		if out == outcomeFailed && !w.hasResolvedArtwork(ctx, item) {
			writeAbsent(ctx, w.deps.ds.Artwork(ctx), item)
		}
		if err := queue.DeleteIfUnchanged(item.ItemKind, item.ItemID, item.ImageType, item.RetryAt); err != nil {
			log.Warn(ctx, "artwork: could not remove exhausted queue item", "kind", item.ItemKind, "id", item.ItemID, err)
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
	ia, err := w.deps.ds.Artwork(ctx).GetItemArtwork(kind, item.ItemID, item.ImageType)
	return err == nil && ia.Hash != ""
}

// precache warms the resize cache at the UI cover size from the bytes just acquired, so the
// first UI request is a cache hit without re-reading the rows or the file. Skipped when
// disabled; failures are debug-only.
func (w *Worker) precache(ctx context.Context, got *acquired) {
	if !conf.Server.EnableArtworkPrecache || w.deps.cache == nil || w.deps.cache.Disabled(ctx) {
		return
	}
	// Same key as the serving path (hash/size/square); only the source of the bytes differs.
	item := &resizedItem{
		hash:       got.ia.Hash,
		size:       conf.Server.UICoverArtSize,
		lastUpdate: got.ia.UpdatedAt,
		ffmpeg:     w.deps.ffmpeg,
		open:       func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(got.data)), nil },
	}
	stream, err := w.deps.cache.Get(ctx, item)
	if err != nil {
		log.Debug(ctx, "artwork: precache failed", "kind", got.ia.ItemKind, "id", got.ia.ItemID, err)
		return
	}
	_, _ = io.Copy(io.Discard, stream)
	_ = stream.Close()
}

// claim reserves items not already in flight, so a row appearing twice within a single
// batch is processed once.
func (w *Worker) claim(batch []model.ArtworkQueueItem) []model.ArtworkQueueItem {
	w.mu.Lock()
	defer w.mu.Unlock()
	var out []model.ArtworkQueueItem
	for _, it := range batch {
		k := queueKey(it)
		if _, busy := w.inFlight[k]; busy {
			continue
		}
		w.inFlight[k] = struct{}{}
		out = append(out, it)
	}
	return out
}

func (w *Worker) release(it model.ArtworkQueueItem) {
	w.mu.Lock()
	delete(w.inFlight, queueKey(it))
	w.mu.Unlock()
}

func queueKey(it model.ArtworkQueueItem) string {
	return it.ItemKind + "|" + it.ItemID + "|" + it.ImageType
}

// gate wraps a named external step with that agent's own rate limiter and circuit
// breaker, matching gateFunc so it can be injected via workerDeps.gate.
func (w *Worker) gate(name string, f func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error) {
	g := w.gateFor(name)
	if !g.breaker.allow() {
		return nil, "", errBreakerOpen
	}
	if err := g.limiter.Wait(w.runCtx); err != nil {
		return nil, "", err
	}
	r, path, err := f()
	g.breaker.record(err)
	return r, path, err
}

// gateFor lazily creates the per-name gate on first use, each with its own limiter at
// ArtworkExternalMaxRPS and its own breaker.
func (w *Worker) gateFor(name string) *extGate {
	w.gatesMu.Lock()
	defer w.gatesMu.Unlock()
	if g, ok := w.gates[name]; ok {
		return g
	}
	rps := conf.Server.ArtworkExternalMaxRPS
	limit := rate.Inf
	if rps > 0 {
		limit = rate.Limit(rps)
	}
	g := &extGate{limiter: rate.NewLimiter(limit, max(1, rps)), breaker: newBreaker()}
	w.gates[name] = g
	return g
}

// backoffFor returns min(5s×4^n, giveUpAfter) scaled by (1+jitter), with jitter in [-0.4, 0.4].
func backoffFor(attempts int, jitter float64) time.Duration {
	d := math.Min(float64(backoffBase)*math.Pow(4, float64(attempts)), float64(giveUpAfter))
	return time.Duration(d * (1 + jitter))
}

func backoff(attempts int) time.Duration {
	return backoffFor(attempts, rand.Float64()*0.8-0.4) //nolint:gosec // retry jitter, not security-sensitive
}

// breaker opens after breakerThreshold consecutive external errors and admits a
// single probe once breakerProbeAfter has elapsed; a success re-closes it.
type breaker struct {
	mu       sync.Mutex
	failures int
	openedAt time.Time
}

func newBreaker() *breaker { return &breaker{} }

func (b *breaker) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failures < breakerThreshold {
		return true
	}
	if time.Since(b.openedAt) >= breakerProbeAfter {
		b.openedAt = time.Now() // start a fresh probe window so only one caller passes
		return true
	}
	return false
}

func (b *breaker) record(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// A not-found (from either package) is a definitive answer, not a fault; only real
	// errors trip the breaker. Must stay consistent with isTransientExternal.
	if err == nil || errors.Is(err, model.ErrNotFound) || errors.Is(err, agents.ErrNotFound) {
		b.failures = 0
		return
	}
	b.failures++
	if b.failures == breakerThreshold {
		b.openedAt = time.Now()
	}
}

package artwork

import (
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
	"golang.org/x/time/rate"
)

const (
	workerPollInterval = 5 * time.Second
	backoffBase        = 5 * time.Minute
	backoffCap         = 48 * time.Hour
	breakerThreshold   = 5
	breakerProbeAfter  = time.Minute
)

var errBreakerOpen = errors.New("artwork: external circuit breaker open")

// extGate is one agent's rate limiter + circuit breaker; each external agent gets its
// own so a provider whose API or CDN is down backs off in isolation from the others.
type extGate struct {
	limiter *rate.Limiter
	breaker *breaker
}

// Worker drains the artwork queue through processItem: each external agent is rate-limited
// and circuit-broken independently, and prune is serialized against in-flight acquisitions
// via pruneMu.
type Worker struct {
	deps    workerDeps
	pruneMu sync.RWMutex
	wake    chan struct{}
	runCtx  context.Context

	gatesMu sync.Mutex
	gates   map[string]*extGate

	mu       sync.Mutex
	inFlight map[string]struct{}
}

func NewWorker(ds model.DataStore, store *ImageStore, ag *agents.Agents, ffmpeg ffmpeg.FFmpeg) *Worker {
	w := &Worker{
		deps:     workerDeps{ds: ds, store: store, agents: ag, ffmpeg: ffmpeg},
		wake:     make(chan struct{}, 1),
		runCtx:   context.Background(),
		gates:    map[string]*extGate{},
		inFlight: map[string]struct{}{},
	}
	w.deps.gate = w.gate
	return w
}

// Run blocks draining the queue until ctx is cancelled. It exits cleanly with no
// leaked goroutines: each drain waits for its batch before the loop can return.
func (w *Worker) Run(ctx context.Context) error {
	w.runCtx = ctx
	concurrency := max(1, conf.Server.ArtworkWorkerConcurrency)
	ticker := time.NewTicker(workerPollInterval)
	defer ticker.Stop()
	for {
		n, err := w.drain(ctx, concurrency)
		if err != nil && ctx.Err() == nil {
			log.Warn(ctx, "artwork: worker drain failed", err)
		}
		if ctx.Err() != nil {
			return nil
		}
		if n > 0 {
			continue // keep draining while the queue has ready work
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		case <-w.wake:
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
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

// RunPrune runs Prune under the worker's write lock, so no acquisition can place
// a file while orphans are being reclaimed. This is the only sanctioned prune path.
func (w *Worker) RunPrune(ctx context.Context) error {
	w.pruneMu.Lock()
	defer w.pruneMu.Unlock()
	return Prune(ctx, w.deps.ds, w.deps.store)
}

func (w *Worker) drain(ctx context.Context, concurrency int) (int, error) {
	// Resolved per drain, not once in Run: the worker starts at boot, possibly before any
	// admin exists, so a late-created admin is picked up on the next poll (private playlists).
	ctx = auth.WithAdminUser(ctx, w.deps.ds)
	batch, err := w.deps.ds.ArtworkQueue(ctx).DequeueBatch(2 * concurrency)
	if err != nil {
		return 0, err
	}
	items := w.claim(batch)
	if len(items) == 0 {
		return 0, nil
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for _, item := range items {
		sem <- struct{}{}
		wg.Add(1)
		go func(it model.ArtworkQueueItem) {
			defer wg.Done()
			defer func() { <-sem }()
			defer w.release(it)
			w.process(ctx, it)
		}(item)
	}
	wg.Wait()
	return len(items), nil
}

func (w *Worker) process(ctx context.Context, item model.ArtworkQueueItem) {
	if item.ImageType == "" {
		item.ImageType = model.ImageTypePrimary
	}
	w.pruneMu.RLock()
	out := processItem(ctx, &w.deps, item)
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
		// MarkFailedIfUnchanged, not MarkFailed: a scan that re-enqueued this row mid-flight reset
		// retry_at, so stale backoff must not stomp its fresh, immediate eligibility.
		retryAt := time.Now().Add(backoff(item.Attempts))
		if err := queue.MarkFailedIfUnchanged(item.ItemKind, item.ItemID, item.ImageType, item.RetryAt, retryAt); err != nil {
			log.Warn(ctx, "artwork: could not reschedule failed queue item", "kind", item.ItemKind, "id", item.ItemID, err)
		}
	}
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

// backoffFor returns min(5m×4^n, 48h) scaled by (1+jitter), with jitter in [-0.2, 0.2].
func backoffFor(attempts int, jitter float64) time.Duration {
	d := math.Min(float64(backoffBase)*math.Pow(4, float64(attempts)), float64(backoffCap))
	return time.Duration(d * (1 + jitter))
}

func backoff(attempts int) time.Duration {
	return backoffFor(attempts, rand.Float64()*0.4-0.2) //nolint:gosec // retry jitter, not security-sensitive
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

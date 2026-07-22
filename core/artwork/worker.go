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
	"github.com/navidrome/navidrome/core/external"
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

// Worker drains the artwork queue and runs each item through processItem. The
// external step is rate-limited and circuit-broken; prune is serialized against
// in-flight acquisitions via pruneMu (acquisitions RLock, prune Lock).
type Worker struct {
	deps    workerDeps
	limiter *rate.Limiter
	breaker *breaker
	pruneMu sync.RWMutex
	wake    chan struct{}
	runCtx  context.Context

	mu       sync.Mutex
	inFlight map[string]struct{}
}

func NewWorker(ds model.DataStore, store *ImageStore, prov external.Provider, ffmpeg ffmpeg.FFmpeg) *Worker {
	rps := conf.Server.DevArtworkExternalRPS
	limit := rate.Inf
	if rps > 0 {
		limit = rate.Limit(rps)
	}
	w := &Worker{
		deps:     workerDeps{ds: ds, store: store, prov: prov, ffmpeg: ffmpeg},
		limiter:  rate.NewLimiter(limit, max(1, rps)),
		breaker:  newBreaker(),
		wake:     make(chan struct{}, 1),
		runCtx:   context.Background(),
		inFlight: map[string]struct{}{},
	}
	w.deps.extGate = w.gate
	return w
}

// Run blocks draining the queue until ctx is cancelled. It exits cleanly with no
// leaked goroutines: each drain waits for its batch before the loop can return.
func (w *Worker) Run(ctx context.Context) error {
	w.runCtx = ctx
	concurrency := max(1, conf.Server.DevArtworkWorkerConcurrency)
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
		if err := queue.Delete(item.ItemKind, item.ItemID, item.ImageType); err != nil {
			log.Warn(ctx, "artwork: could not delete processed queue item", "kind", item.ItemKind, "id", item.ItemID, err)
		}
	case outcomeFailed:
		retryAt := time.Now().Add(backoff(item.Attempts))
		if err := queue.MarkFailed(item.ItemKind, item.ItemID, item.ImageType, retryAt); err != nil {
			log.Warn(ctx, "artwork: could not reschedule failed queue item", "kind", item.ItemKind, "id", item.ItemID, err)
		}
	}
}

// claim reserves items not already in flight, so a wake-triggered re-drain never
// double-processes an item still running from a previous cycle.
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

// gate wraps the external step with the rate limiter and circuit breaker, matching
// extGateFunc so it can be injected via workerDeps.extGate.
func (w *Worker) gate(f func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error) {
	if !w.breaker.allow() {
		return nil, "", errBreakerOpen
	}
	if err := w.limiter.Wait(w.runCtx); err != nil {
		return nil, "", err
	}
	r, path, err := f()
	w.breaker.record(err)
	return r, path, err
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
	// A not-found is a definitive answer, not a fault; only real errors trip the breaker.
	if err == nil || errors.Is(err, model.ErrNotFound) {
		b.failures = 0
		return
	}
	b.failures++
	if b.failures == breakerThreshold {
		b.openedAt = time.Now()
	}
}

package artwork

import (
	"errors"
	"io"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"golang.org/x/time/rate"
)

const (
	breakerThreshold  = 5
	breakerProbeAfter = time.Minute
)

var errBreakerOpen = errors.New("artwork: external circuit breaker open")

// gateFunc gates one named external fetch (rate limit + circuit breaker per name).
type gateFunc = func(name string, f func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error)

func passthroughGate(_ string, f func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error) {
	return f()
}

// isTransientExternal reports whether an external failure is worth retrying; a not-found
// (from either package) is a definitive answer, not a fault.
func isTransientExternal(err error) bool {
	return err != nil && !errors.Is(err, agents.ErrNotFound) && !errors.Is(err, model.ErrNotFound)
}

// extGate is one agent's rate limiter and circuit breaker, so a failing provider backs off
// in isolation from the others.
type extGate struct {
	limiter *rate.Limiter
	breaker *breaker
}

// gate runs a named external step through that agent's rate limiter and circuit breaker.
func (w *Worker) gate(name string, f func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error) {
	g := w.gateFor(name)
	if !g.breaker.allow() {
		log.Debug(w.runCtx, "Artwork: Skipping agent, circuit breaker open", "agent", name)
		return nil, "", errBreakerOpen
	}
	// Timed separately so a throttled agent isn't mistaken for a slow provider.
	waitStart := time.Now()
	if err := g.limiter.Wait(w.runCtx); err != nil {
		return nil, "", err
	}
	callStart := time.Now()
	r, path, err := f()
	g.breaker.record(name, err)
	log.Trace(w.runCtx, "Artwork: External agent call", "agent", name, "hit", r != nil,
		"limiterWait", callStart.Sub(waitStart), "elapsed", time.Since(callStart), err)
	return r, path, err
}

// gateFor lazily creates the per-name gate on first use.
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

// breaker opens after breakerThreshold consecutive errors and admits a single probe once
// breakerProbeAfter has elapsed; a success re-closes it.
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

func (b *breaker) record(name string, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// A not-found is a definitive answer, not a fault; keep in sync with isTransientExternal.
	if err == nil || errors.Is(err, model.ErrNotFound) || errors.Is(err, agents.ErrNotFound) {
		if b.failures >= breakerThreshold {
			log.Info("Artwork: Circuit breaker closed for agent", "agent", name)
		}
		b.failures = 0
		return
	}
	b.failures++
	if b.failures == breakerThreshold {
		b.openedAt = time.Now()
		log.Warn("Artwork: Circuit breaker opened for agent", "agent", name,
			"consecutiveFailures", b.failures, "probeAfter", breakerProbeAfter, err)
	}
}

package artwork

import (
	"context"
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
	// breakerRecoveries is how many consecutive answers an open breaker needs before it trusts the
	// provider again. One is not enough: a provider that is rate-limiting or blocking us still
	// answers the occasional request, and closing on the first of those puts the agent straight
	// back to full rate, which is what earns the next block.
	breakerRecoveries = 3
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
	allowed, gen := g.breaker.allow()
	if !allowed {
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
	g.breaker.record(name, gen, err)
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
	rps := conf.Server.DevArtworkExternalMaxRPS
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
	// recoveries counts consecutive good answers while open; a single failure discards them.
	recoveries int
	// generation identifies the current open episode, so an answer from a call admitted before
	// the breaker opened cannot be mistaken for evidence that it has recovered.
	generation int
}

func newBreaker() *breaker { return &breaker{} }

// allow reports whether a call may proceed, and the open episode it was admitted under: zero
// when the breaker was closed, the current generation when admitted as a half-open probe.
func (b *breaker) allow() (bool, int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failures < breakerThreshold {
		return true, 0
	}
	if time.Since(b.openedAt) >= breakerProbeAfter {
		b.openedAt = time.Now() // start a fresh probe window so only one caller passes
		return true, b.generation
	}
	return false, 0
}

func (b *breaker) record(name string, gen int, err error) {
	// A cancelled run says nothing about the provider, so it neither counts nor clears.
	if errors.Is(err, context.Canceled) {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if isTransientExternal(err) {
		b.recoveries = 0
		b.failures++
		if b.failures == breakerThreshold {
			b.openedAt = time.Now()
			b.generation++
			log.Warn("Artwork: Circuit breaker opened for agent", "agent", name,
				"consecutiveFailures", b.failures, "probeAfter", breakerProbeAfter, err)
		}
		return
	}
	if b.failures < breakerThreshold {
		b.failures = 0
		return
	}
	// Only a probe from this open episode is evidence of recovery. The worker drains concurrently,
	// so answers keep arriving from calls admitted before the breaker opened; counting those would
	// close it with no probe interval elapsed, which is the burst this exists to prevent.
	if gen == 0 || gen != b.generation {
		return
	}
	// A not-found counts because the provider did answer, but on its own it is thin evidence that
	// a provider which just blocked us is well.
	b.recoveries++
	if b.recoveries < breakerRecoveries {
		return
	}
	log.Info("Artwork: Circuit breaker closed for agent", "agent", name,
		"consecutiveAnswers", b.recoveries)
	b.failures, b.recoveries = 0, 0
}

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
// resolveItem defaults to passthroughGate; the worker injects the per-agent gate.
type gateFunc = func(name string, f func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error)

func passthroughGate(_ string, f func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error) {
	return f()
}

// isTransientExternal reports whether an external step failed in a way worth retrying;
// a not-found (from either package) is a definitive answer, not a fault.
func isTransientExternal(err error) bool {
	return err != nil && !errors.Is(err, agents.ErrNotFound) && !errors.Is(err, model.ErrNotFound)
}

// extGate is one agent's rate limiter + circuit breaker; each external agent gets its
// own so a provider whose API or CDN is down backs off in isolation from the others.
type extGate struct {
	limiter *rate.Limiter
	breaker *breaker
}

// gate wraps a named external step with that agent's own rate limiter and circuit
// breaker, matching gateFunc so it can be handed to the processor's resolver.
func (w *Worker) gate(name string, f func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error) {
	g := w.gateFor(name)
	if !g.breaker.allow() {
		log.Debug(w.runCtx, "Artwork: Skipping agent, circuit breaker open", "agent", name)
		return nil, "", errBreakerOpen
	}
	// Waiting for the rate-limit permit is counted separately: a slow agent and a throttled one
	// look identical from the drain, but only one of them is the provider's fault.
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

func (b *breaker) record(name string, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// A not-found (from either package) is a definitive answer, not a fault; only real
	// errors trip the breaker. Must stay consistent with isTransientExternal.
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

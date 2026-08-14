package artwork

import (
	"context"
	"slices"
	"sync"
)

const (
	outcomeHit        = "hit"
	outcomeMiss       = "miss"
	outcomeUnreadable = "unreadable"
	outcomeSkipped    = "skipped"
	outcomeWouldTry   = "would-try"
	outcomeError      = "error"
	outcomeNotReached = "not-reached"
)

// traceStep is one candidate the priority chain considered.
type traceStep struct {
	Candidate string
	Outcome   string
	Detail    string
}

// chainTrace collects the walk of a single resolution. The artwork worker never attaches
// one; only the CLI does, so resolution stays allocation-free in the hot path.
type chainTrace struct {
	mu    sync.Mutex
	steps []traceStep
}

func (t *chainTrace) add(step traceStep) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.steps = append(t.steps, step)
}

func (t *chainTrace) Steps() []traceStep {
	t.mu.Lock()
	defer t.mu.Unlock()
	return slices.Clone(t.steps)
}

type traceCtxKey struct{}

func withTrace(ctx context.Context, t *chainTrace) context.Context {
	return context.WithValue(ctx, traceCtxKey{}, t)
}

func traceFrom(ctx context.Context) *chainTrace {
	t, _ := ctx.Value(traceCtxKey{}).(*chainTrace)
	return t
}

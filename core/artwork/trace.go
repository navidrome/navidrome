package artwork

import (
	"context"
	"errors"
	"io"
	"slices"
	"sync"
)

// Outcome is what the priority chain observed for one candidate. The CLI branches on these,
// so they are part of the package's API, not private labels.
type Outcome string

const (
	OutcomeHit        Outcome = "hit"
	OutcomeMiss       Outcome = "miss"
	OutcomeUnreadable Outcome = "unreadable"
	OutcomeSkipped    Outcome = "skipped"
	OutcomeWouldTry   Outcome = "would-try"
	OutcomeError      Outcome = "error"
)

const (
	// ExternalCandidate labels the external tier itself, for the cases that never reach an agent.
	ExternalCandidate = "external"
	// ExternalPrefix qualifies a candidate or a stored source with the agent that produced it.
	ExternalPrefix = ExternalCandidate + ":"
)

// TraceStep is one candidate the priority chain considered.
type TraceStep struct {
	Candidate string
	Outcome   Outcome
	Detail    string
}

// ChainTrace collects the walk of a single resolution. The artwork worker never attaches
// one; only the CLI does, so resolution stays allocation-free in the hot path.
type ChainTrace struct {
	mu    sync.Mutex
	steps []TraceStep
}

func (t *ChainTrace) add(step TraceStep) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.steps = append(t.steps, step)
}

func (t *ChainTrace) Steps() []TraceStep {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return slices.Clone(t.steps)
}

type traceCtxKey struct{}

func withTrace(ctx context.Context, t *ChainTrace) context.Context {
	return context.WithValue(ctx, traceCtxKey{}, t)
}

func traceFrom(ctx context.Context) *ChainTrace {
	t, _ := ctx.Value(traceCtxKey{}).(*ChainTrace)
	return t
}

var errOfflineSkipped = errors.New("artwork: external lookup skipped (offline)")

// tracingGate records each external agent's outcome without changing what the gate returns.
func tracingGate(t *ChainTrace, inner gateFunc) gateFunc {
	return func(name string, f func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error) {
		r, path, err := inner(name, f)
		candidate := ExternalPrefix + name
		switch {
		case r != nil:
			t.add(TraceStep{Candidate: candidate, Outcome: OutcomeHit, Detail: path})
		case isTransientExternal(err):
			t.add(TraceStep{Candidate: candidate, Outcome: OutcomeError, Detail: err.Error()})
		default:
			t.add(TraceStep{Candidate: candidate, Outcome: OutcomeMiss})
		}
		return r, path, err
	}
}

// offlineGate reports which agents would be asked without asking them, so a diagnostic
// command cannot add load to a provider that is already rate-limiting us.
func offlineGate(t *ChainTrace) gateFunc {
	return func(name string, _ func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error) {
		t.add(TraceStep{Candidate: ExternalPrefix + name, Outcome: OutcomeWouldTry})
		return nil, "", errOfflineSkipped
	}
}

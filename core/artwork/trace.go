package artwork

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"sync"

	"github.com/navidrome/navidrome/utils/str"
)

// Outcome is what the priority chain observed for one candidate; the CLI renders and branches on these.
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
	// externalCandidate labels the external tier itself, for the cases that never reach an agent.
	externalCandidate = "external"
	// ExternalPrefix qualifies a candidate or a stored source with the agent that produced it.
	ExternalPrefix = externalCandidate + ":"
)

// TraceStep is one candidate the priority chain considered.
type TraceStep struct {
	Candidate string
	Outcome   Outcome
	Detail    string
}

// ChainTrace collects the walk of a single resolution: the worker attaches one per queue
// item so it can be stored, and the CLI attaches one per explain.
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

// maxTraceDetail bounds a stored Detail, which on the failure paths is an error string of
// unknown length. Past ~1kB a row spills to an overflow page, slowing every scan of the table.
const maxTraceDetail = 200

// storedStep is the persisted shape of a TraceStep. The keys are single letters because a trace
// is written for every item, and the encoded length is repeated across the whole library.
type storedStep struct {
	C string  `json:"c"`
	O Outcome `json:"o"`
	D string  `json:"d,omitempty"`
}

// EncodeTrace serializes steps for storage. A hit's Detail is the winning source's path, which
// the same row already stores as source_path, so it is dropped and DecodeTrace puts it back.
func EncodeTrace(steps []TraceStep, sourcePath string) string {
	out := make([]storedStep, 0, len(steps))
	for _, s := range steps {
		d := s.Detail
		if s.Outcome == OutcomeHit && d == sourcePath {
			d = ""
		}
		out = append(out, storedStep{C: s.Candidate, O: s.Outcome, D: str.TruncateRunes(d, maxTraceDetail, "...")})
	}
	b, _ := json.Marshal(out) // []storedStep is all strings, so this cannot fail
	return string(b)
}

// DecodeTrace reverses EncodeTrace. A trace that will not parse is reported as no trace at all,
// since a diagnostic command must not fail on a bad row.
func DecodeTrace(encoded, sourcePath string) []TraceStep {
	if encoded == "" {
		return nil
	}
	var stored []storedStep
	if err := json.Unmarshal([]byte(encoded), &stored); err != nil {
		return nil
	}
	steps := make([]TraceStep, 0, len(stored))
	for _, s := range stored {
		d := s.D
		if d == "" && s.O == OutcomeHit {
			d = sourcePath
		}
		steps = append(steps, TraceStep{Candidate: s.C, Outcome: s.O, Detail: d})
	}
	return steps
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

// recordAgent files what one external agent answered. The agent loops call this rather than a
// gate wrapper, because only they hold the context that carries the trace.
func recordAgent(ctx context.Context, name string, r io.ReadCloser, path string, err error) {
	t := traceFrom(ctx)
	candidate := ExternalPrefix + name
	switch {
	case r != nil:
		t.add(TraceStep{Candidate: candidate, Outcome: OutcomeHit, Detail: path})
	case errors.Is(err, errOfflineSkipped):
		t.add(TraceStep{Candidate: candidate, Outcome: OutcomeWouldTry})
	case isTransientExternal(err):
		t.add(TraceStep{Candidate: candidate, Outcome: OutcomeError, Detail: err.Error()})
	default:
		t.add(TraceStep{Candidate: candidate, Outcome: OutcomeMiss})
	}
}

// traceStage records a failure from the stages that run after the priority chain has already
// picked a winner: most ways an item can fail are here, not in the chain walk.
func traceStage(ctx context.Context, stage string, err error) {
	traceFrom(ctx).add(TraceStep{Candidate: stage, Outcome: OutcomeError, Detail: err.Error()})
}

// offlineGate reports which agents would be asked without asking them, so a diagnostic
// command cannot add load to a provider that is already rate-limiting us.
func offlineGate(string, func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error) {
	return nil, "", errOfflineSkipped
}

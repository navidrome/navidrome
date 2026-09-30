package plugins

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"sync"
)

const maxOpenStreams = 64

var (
	errUnknownStream          = errors.New("unknown stream handle")
	errTooManyStreams         = errors.New("too many open streams")
	errStreamHandlesExhausted = errors.New("stream handles exhausted")
	errCallEnded              = errors.New("plugin call has ended")
	errPluginUnloaded         = errors.New("plugin is unloaded")
	errDuplicateStream        = errors.New("stream handle returned more than once")
)

// streamTable holds the stream handles of one loaded plugin. One mutex guards the table and all its scopes,
// so scope state changes are linearized with registration and transfer.
type streamTable struct {
	mu     sync.Mutex
	next   uint32
	open   int
	scopes map[*callScope]struct{}
	closed bool
}

func newStreamTable() *streamTable {
	return &streamTable{scopes: map[*callScope]struct{}{}}
}

// callScope owns the handles of one plugin call. baseCtx is the caller's ctx from before the call deadline and
// Extism wrapped it: anything that must outlive the call derives from it.
type callScope struct {
	table   *streamTable
	ctx     context.Context //nolint:containedctx // checked on every mutation
	baseCtx context.Context //nolint:containedctx // parent for work that outlives the call
	closed  bool
	done    chan struct{} // closed once every stream of the scope is closed
	entries map[uint32]*streamBackend
	lastErr error        // only touched by the call's goroutine, through host functions
	length  kernelLength // like lastErr
}

func (t *streamTable) newScope(ctx, baseCtx context.Context) (*callScope, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil, errPluginUnloaded
	}
	s := &callScope{table: t, ctx: ctx, baseCtx: baseCtx, done: make(chan struct{}), entries: map[uint32]*streamBackend{}}
	t.scopes[s] = struct{}{}
	return s, nil
}

// Close closes every live scope; it runs when the plugin unloads.
func (t *streamTable) Close() error {
	t.mu.Lock()
	t.closed = true
	scopes := slices.Collect(maps.Keys(t.scopes))
	t.mu.Unlock()
	for _, s := range scopes {
		s.close()
	}
	return nil
}

// checkOpenLocked checks ctx before the closed flag, so a cancel reports the same error whether or not its
// AfterFunc has closed the scope yet.
func (s *callScope) checkOpenLocked() error {
	if s.table.closed {
		return errPluginUnloaded
	}
	if err := s.ctx.Err(); err != nil {
		return err
	}
	if s.closed {
		return errCallEnded
	}
	return nil
}

func (s *callScope) register(b *streamBackend) (uint32, error) {
	h, err := s.insert(b)
	if err != nil {
		_ = b.Close()
	}
	return h, err
}

func (s *callScope) insert(b *streamBackend) (uint32, error) {
	t := s.table
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := s.checkOpenLocked(); err != nil {
		return 0, err
	}
	if t.open >= maxOpenStreams {
		return 0, errTooManyStreams
	}
	if t.next >= math.MaxInt32 {
		return 0, errStreamHandlesExhausted
	}
	t.next++
	s.entries[t.next] = b
	t.open++
	return t.next, nil
}

func (s *callScope) lookup(h uint32, need streamRights) (*streamBackend, error) {
	s.table.mu.Lock()
	b, ok := s.entries[h]
	s.table.mu.Unlock()
	switch {
	case !ok:
		return nil, errUnknownStream
	case need&streamRead != 0 && b.rights&streamRead == 0:
		return nil, errStreamNotReadable
	case need&streamWrite != 0 && b.rights&streamWrite == 0:
		return nil, errStreamNotWritable
	}
	return b, nil
}

func (s *callScope) detach(h uint32) (*streamBackend, error) {
	s.table.mu.Lock()
	defer s.table.mu.Unlock()
	b, ok := s.entries[h]
	if !ok {
		return nil, errUnknownStream
	}
	delete(s.entries, h)
	s.table.open--
	return b, nil
}

// transfer moves the handles out of the scope to the host caller, all or nothing.
func (s *callScope) transfer(handles ...uint32) ([]any, error) {
	t := s.table
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := s.checkOpenLocked(); err != nil {
		return nil, err
	}
	seen := make(map[uint32]bool, len(handles))
	for _, h := range handles {
		if seen[h] {
			return nil, errDuplicateStream
		}
		seen[h] = true
		if _, ok := s.entries[h]; !ok {
			return nil, fmt.Errorf("%w: %d", errUnknownStream, h)
		}
	}
	values := make([]any, len(handles))
	for i, h := range handles {
		values[i] = s.entries[h].transferValue()
		delete(s.entries, h)
		t.open--
	}
	return values, nil
}

// close returns only after every stream of the scope is closed, even when another caller started closing it.
func (s *callScope) close() {
	t := s.table
	t.mu.Lock()
	if s.closed {
		t.mu.Unlock()
		<-s.done
		return
	}
	s.closed = true
	backends := slices.Collect(maps.Values(s.entries))
	t.open -= len(backends)
	s.entries = nil
	t.mu.Unlock()

	for _, b := range backends {
		_ = b.Close()
	}

	t.mu.Lock()
	delete(t.scopes, s)
	t.mu.Unlock()
	close(s.done)
}

type callScopeKey struct{}

func withCallScope(ctx context.Context, s *callScope) context.Context {
	return context.WithValue(ctx, callScopeKey{}, s)
}

func callScopeFrom(ctx context.Context) *callScope {
	s, _ := ctx.Value(callScopeKey{}).(*callScope)
	return s
}

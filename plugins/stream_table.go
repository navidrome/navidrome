package plugins

import (
	"context"
	"errors"
	"fmt"
	"io"
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

// callScope owns the handles of one plugin call. ctxErr is the call ctx's Err.
type callScope struct {
	table   *streamTable
	ctxErr  func() error
	closed  bool
	done    chan struct{} // closed once every stream of the scope is closed
	entries map[uint32]*streamBackend
	lastErr error        // only touched by the call's goroutine, through host functions
	length  kernelLength // like lastErr
}

func (t *streamTable) newScope(ctx context.Context) (*callScope, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil, errPluginUnloaded
	}
	s := &callScope{table: t, ctxErr: ctx.Err, done: make(chan struct{}), entries: map[uint32]*streamBackend{}}
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
	if err := s.ctxErr(); err != nil {
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
	err := s.checkOpenLocked()
	b, ok := s.entries[h]
	s.table.mu.Unlock()
	switch {
	case err != nil:
		return nil, err
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
	if err := s.checkOpenLocked(); err != nil {
		return nil, err
	}
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

type streamRights uint8

const (
	streamRead streamRights = 1 << iota
	streamWrite
)

// maxEmptyReads bounds how often a Read may return (0, nil); io.Copy would otherwise loop forever.
const maxEmptyReads = 100

var (
	errStreamNotReadable = errors.New("stream is not readable")
	errStreamNotWritable = errors.New("stream is not writable")
)

// streamBackend is what a handle points to. It keeps an error that arrived together with data, so every later
// consumer (plugin reads, stream_copy, host code after transfer) still sees it.
type streamBackend struct {
	r      io.Reader
	w      io.Writer
	c      io.Closer
	rights streamRights

	mu       sync.Mutex
	retained error
	eof      bool

	closeOnce sync.Once
	closeErr  error
}

func newReadStream(rc io.ReadCloser) *streamBackend {
	return &streamBackend{r: rc, c: rc, rights: streamRead}
}

func newWriteStream(wc io.WriteCloser) *streamBackend {
	return &streamBackend{w: wc, c: wc, rights: streamWrite}
}

func (b *streamBackend) Read(p []byte) (int, error) {
	if err := b.readErr(); err != nil {
		return 0, err
	}
	if b.r == nil {
		return 0, errStreamNotReadable
	}
	if len(p) == 0 {
		return 0, nil
	}
	for range maxEmptyReads {
		n, err := b.r.Read(p)
		if errors.Is(err, io.EOF) {
			b.setEOF()
			if n > 0 {
				return n, nil
			}
			return 0, io.EOF
		}
		if err != nil {
			b.retain(err)
			if n > 0 {
				return n, nil
			}
			return 0, err
		}
		if n > 0 {
			return n, nil
		}
	}
	b.retain(io.ErrNoProgress)
	return 0, io.ErrNoProgress
}

func (b *streamBackend) Write(p []byte) (int, error) {
	if err := b.retainedErr(); err != nil {
		return 0, err
	}
	if b.w == nil {
		return 0, errStreamNotWritable
	}
	n, err := b.w.Write(p)
	if err == nil && n < len(p) {
		err = io.ErrShortWrite
	}
	if err != nil {
		b.retain(err)
	}
	return n, err
}

func (b *streamBackend) Close() error {
	b.closeOnce.Do(func() {
		b.closeErr = b.c.Close()
		// A writer may accept every byte and still fail; the caller saw a full count, so Close reports it.
		if b.closeErr == nil && b.w != nil {
			b.closeErr = b.retainedErr()
		}
	})
	return b.closeErr
}

// transferValue is what host code gets when a plugin returns this stream: the original value keeps abilities
// like io.Seeker, unless a retained error would be lost with it.
func (b *streamBackend) transferValue() any {
	if b.retainedErr() != nil {
		return b
	}
	return b.c
}

func (b *streamBackend) retain(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.retained == nil {
		b.retained = err
	}
}

func (b *streamBackend) setEOF() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.eof = true
}

func (b *streamBackend) readErr() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.eof {
		return io.EOF
	}
	return b.retained
}

func (b *streamBackend) retainedErr() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.retained
}

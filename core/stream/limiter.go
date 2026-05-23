package stream

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
)

// ErrTooManyTranscodes is returned by TranscodeLimiter.Acquire when the
// configured concurrency cap has been reached. Callers should translate this
// into an HTTP 429 response so well-behaved clients back off and retry.
var ErrTooManyTranscodes = errors.New("too many concurrent transcodes")

// TranscodeLimiter gates the number of concurrent ffmpeg transcodes. It enforces
// both a global cap (to protect the host from process exhaustion) and an optional
// per-user cap (to keep one client from starving the others). Acquire never
// blocks: it either reserves a slot or returns ErrTooManyTranscodes immediately.
type TranscodeLimiter interface {
	// Acquire reserves a slot for the given user. On success it returns a release
	// function that must be called exactly once when the transcode is done.
	// Calling release more than once is safe and idempotent.
	Acquire(ctx context.Context, user string) (release func(), err error)
}

// NewTranscodeLimiter returns a limiter enforcing the given caps. Each cap is
// independent: a value of zero or less disables that cap. When both caps are
// disabled the limiter is a no-op.
func NewTranscodeLimiter(maxConcurrent, maxPerUser int) TranscodeLimiter {
	if maxConcurrent <= 0 && maxPerUser <= 0 {
		return noopLimiter{}
	}
	l := &transcodeLimiter{
		maxConcurrent: maxConcurrent,
		maxPerUser:    maxPerUser,
		perUser:       make(map[string]int),
	}
	if maxConcurrent > 0 {
		l.global = make(chan struct{}, maxConcurrent)
	}
	return l
}

// releasingReadCloser wraps an io.ReadCloser so that closing it also releases
// the limiter slot exactly once. release must be the function returned by
// TranscodeLimiter.Acquire; its own idempotency makes double-Close safe too.
type releasingReadCloser struct {
	io.ReadCloser
	release func()
}

func (r *releasingReadCloser) Close() error {
	err := r.ReadCloser.Close()
	r.release()
	return err
}

type noopLimiter struct{}

func (noopLimiter) Acquire(context.Context, string) (func(), error) {
	return func() {}, nil
}

type transcodeLimiter struct {
	maxConcurrent int
	maxPerUser    int
	global        chan struct{}

	mu      sync.Mutex
	perUser map[string]int
}

func (l *transcodeLimiter) Acquire(_ context.Context, user string) (func(), error) {
	// Reserve a per-user slot first so a noisy user can't burn through
	// global slots only to be rejected later.
	if l.maxPerUser > 0 {
		l.mu.Lock()
		if l.perUser[user] >= l.maxPerUser {
			l.mu.Unlock()
			return nil, ErrTooManyTranscodes
		}
		l.perUser[user]++
		l.mu.Unlock()
	}

	if l.global != nil {
		select {
		case l.global <- struct{}{}:
		default:
			l.releasePerUser(user)
			return nil, ErrTooManyTranscodes
		}
	}

	var released atomic.Bool
	return func() {
		if !released.CompareAndSwap(false, true) {
			return
		}
		if l.global != nil {
			<-l.global
		}
		l.releasePerUser(user)
	}, nil
}

func (l *transcodeLimiter) releasePerUser(user string) {
	if l.maxPerUser <= 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.perUser[user]--
	if l.perUser[user] <= 0 {
		delete(l.perUser, user)
	}
}

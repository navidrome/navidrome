package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
)

var (
	ErrThrottleCapacityExceeded = errors.New("throttle: capacity exceeded")
	ErrThrottleTimeout          = errors.New("throttle: backlog timeout")
)

type requestThrottle struct {
	tokens         chan struct{}
	backlogTokens  chan struct{}
	backlogTimeout time.Duration
}

// ThrottleBacklog creates a Chi-compatible middleware that limits concurrent
// request processing. Unlike Chi's ThrottleBacklog, it buffers the handler's
// response while holding the token, releases it, then flushes the buffer to
// the client with a write deadline. This prevents slow clients from holding
// throttle capacity.
func ThrottleBacklog(limit, backlogLimit int, backlogTimeout time.Duration) func(http.Handler) http.Handler {
	if limit <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	if !conf.Server.DevArtworkThrottleBuffered {
		return middleware.ThrottleBacklog(limit, backlogLimit, backlogTimeout)
	}
	t := &requestThrottle{
		tokens:         make(chan struct{}, limit),
		backlogTokens:  make(chan struct{}, limit+backlogLimit),
		backlogTimeout: backlogTimeout,
	}
	for range limit {
		t.tokens <- struct{}{}
	}
	for range limit + backlogLimit {
		t.backlogTokens <- struct{}{}
	}
	return t.handler
}

func (t *requestThrottle) handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		release, err := t.acquire(ctx)
		if err != nil {
			switch {
			case errors.Is(err, ErrThrottleCapacityExceeded):
				log.Warn(ctx, "Request throttle capacity exceeded", "path", r.URL.Path)
			case errors.Is(err, ErrThrottleTimeout):
				log.Warn(ctx, "Request throttle backlog timeout", "path", r.URL.Path)
			}
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		buf := &bufferedResponseWriter{header: make(http.Header)}
		next.ServeHTTP(buf, r)
		release()

		if err := setWriteTimeout(w, consts.ArtworkWriteTimeout); err != nil {
			log.Debug(ctx, "Could not set write timeout", err)
		}
		for k, v := range buf.header {
			w.Header()[k] = v
		}
		if buf.code > 0 {
			w.WriteHeader(buf.code)
		}
		if _, err := w.Write(buf.body.Bytes()); err != nil {
			log.Warn(ctx, "Error writing throttled response", err)
		}
	})
}

func (t *requestThrottle) acquire(ctx context.Context) (release func(), err error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-t.backlogTokens:
	default:
		return nil, ErrThrottleCapacityExceeded
	}

	select {
	case <-t.tokens:
		return t.releaseFunc(), nil
	default:
	}

	timer := time.NewTimer(t.backlogTimeout)
	select {
	case <-timer.C:
		t.backlogTokens <- struct{}{}
		return nil, ErrThrottleTimeout
	case <-ctx.Done():
		timer.Stop()
		t.backlogTokens <- struct{}{}
		return nil, ctx.Err()
	case <-t.tokens:
		timer.Stop()
		return t.releaseFunc(), nil
	}
}

func (t *requestThrottle) releaseFunc() func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			t.tokens <- struct{}{}
			t.backlogTokens <- struct{}{}
		})
	}
}

type bufferedResponseWriter struct {
	header http.Header
	body   bytes.Buffer
	code   int
}

func (w *bufferedResponseWriter) Header() http.Header {
	return w.header
}

func (w *bufferedResponseWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *bufferedResponseWriter) WriteHeader(code int) {
	w.code = code
}

// setWriteTimeout sets a write deadline on the response writer by walking the
// Unwrap chain to find a writer that supports SetWriteDeadline.
func setWriteTimeout(rw io.Writer, timeout time.Duration) error {
	for {
		switch t := rw.(type) {
		case interface{ SetWriteDeadline(time.Time) error }:
			return t.SetWriteDeadline(time.Now().Add(timeout))
		case interface{ Unwrap() http.ResponseWriter }:
			rw = t.Unwrap()
		default:
			return fmt.Errorf("%T - %w", rw, http.ErrNotSupported)
		}
	}
}

package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
)

var (
	ErrThrottleCapacityExceeded = errors.New("throttle: capacity exceeded")
	ErrThrottleTimeout          = errors.New("throttle: backlog timeout")
)

type RequestThrottle struct {
	tokens         chan struct{}
	backlogTokens  chan struct{}
	backlogTimeout time.Duration
}

func NewRequestThrottle() *RequestThrottle {
	limit := conf.Server.DevArtworkMaxRequests
	if limit <= 0 {
		return nil
	}
	backlogLimit := conf.Server.DevArtworkThrottleBacklogLimit
	backlogTimeout := conf.Server.DevArtworkThrottleBacklogTimeout
	log.Debug("Creating request throttle", "maxRequests", limit,
		"backlogLimit", backlogLimit, "backlogTimeout", backlogTimeout)
	t := &RequestThrottle{
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
	return t
}

func (t *RequestThrottle) acquire(ctx context.Context) (release func(), err error) {
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

func (t *RequestThrottle) releaseFunc() func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			t.tokens <- struct{}{}
			t.backlogTokens <- struct{}{}
		})
	}
}

type ReaderFunc func() (io.ReadCloser, time.Time, error)

// DoBuffered acquires a throttle token, calls fn, buffers the result, and
// releases the token before returning. Safe to call on a nil receiver
// (throttling is skipped).
func (t *RequestThrottle) DoBuffered(ctx context.Context, fn ReaderFunc) (*bytes.Buffer, time.Time, error) {
	if t != nil {
		release, err := t.acquire(ctx)
		if err != nil {
			return nil, time.Time{}, err
		}
		defer release()
	}

	reader, lastUpdate, err := fn()
	if err != nil {
		return nil, time.Time{}, err
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	reader.Close()
	if err != nil {
		return nil, time.Time{}, err
	}
	return &buf, lastUpdate, nil
}

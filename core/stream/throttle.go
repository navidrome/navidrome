package stream

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/singleton"
	"golang.org/x/sync/semaphore"
)

// ErrTranscodingBusy is returned when the transcoding throttle rejects a request
// because the concurrency limit and backlog are both full.
var ErrTranscodingBusy = errors.New("too many concurrent transcodes")

// transcodingThrottle limits the number of concurrent transcoding operations.
type transcodingThrottle struct {
	sem        *semaphore.Weighted
	backlog    atomic.Int64
	maxBacklog int64
	timeout    time.Duration
	disabled   bool
}

func newTranscodingThrottle(maxConcurrent, maxBacklog int, timeout time.Duration) *transcodingThrottle {
	if maxConcurrent <= 0 {
		return &transcodingThrottle{disabled: true}
	}
	return &transcodingThrottle{
		sem:        semaphore.NewWeighted(int64(maxConcurrent)),
		maxBacklog: int64(maxBacklog),
		timeout:    timeout,
	}
}

// Acquire blocks until a transcoding slot is available, the backlog is full, or the timeout expires.
func (t *transcodingThrottle) Acquire(ctx context.Context) error {
	if t.disabled {
		return nil
	}

	// Fast path: try to acquire without touching the backlog counter
	if t.sem.TryAcquire(1) {
		return nil
	}

	// Slow path: semaphore is full, enter backlog queue
	// Increment-then-check-then-rollback to avoid TOCTOU race
	current := t.backlog.Add(1)
	if current > t.maxBacklog {
		t.backlog.Add(-1)
		log.Warn(ctx, "Transcoding request rejected, throttle backlog full", "backlog", current-1)
		return ErrTranscodingBusy
	}

	log.Info(ctx, "Transcoding request queued, waiting for slot", "backlog", current)
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	err := t.sem.Acquire(ctx, 1)
	t.backlog.Add(-1)
	if err != nil {
		log.Warn(ctx, "Transcoding request rejected, timeout waiting for slot")
		return ErrTranscodingBusy
	}
	return nil
}

// Release frees a transcoding slot.
func (t *transcodingThrottle) Release() {
	if t.disabled {
		return
	}
	t.sem.Release(1)
}

// releaseOnClose wraps a ReadCloser to call a release function exactly once on Close.
type releaseOnClose struct {
	io.ReadCloser
	release func()
	once    sync.Once
}

func (r *releaseOnClose) Close() error {
	err := r.ReadCloser.Close()
	r.once.Do(r.release)
	return err
}

// getTranscodingThrottle returns a singleton transcodingThrottle created from the current configuration.
func getTranscodingThrottle() *transcodingThrottle {
	return singleton.GetInstance(func() *transcodingThrottle {
		return newTranscodingThrottle(
			conf.Server.MaxConcurrentTranscodes,
			conf.Server.DevTranscodeThrottleBacklogLimit,
			conf.Server.DevTranscodeThrottleBacklogTimeout,
		)
	})
}

package subsonic

import (
	"cmp"
	"context"
	"hash/maphash"
	"sync"
	"time"

	"github.com/navidrome/navidrome/consts"
)

// authLimiter caps failed Subsonic logins per key. Checks run at most `limit` at a time and failures
// are recorded afterwards, so a window admits up to 2*limit-1 guesses and valid requests only wait.
type authLimiter struct {
	limit     int
	window    time.Duration
	seed      maphash.Seed
	mu        sync.Mutex
	keys      map[uint64]*authAttempts // hashed, so attacker-chosen usernames cannot bloat memory
	lastSweep time.Time
}

type authAttempts struct {
	failures int
	start    time.Time
	slots    chan struct{}
	refs     int
}

// authSlot is a reserved credential check. A nil slot releases nothing, which is what a disabled
// limiter hands back.
type authSlot struct {
	limiter *authLimiter
	entry   *authAttempts
}

// newAuthLimiter returns nil when limit is not positive. A nil limiter allows everything.
func newAuthLimiter(limit int, window time.Duration) *authLimiter {
	if limit <= 0 {
		return nil
	}
	return &authLimiter{
		limit:  limit,
		window: cmp.Or(window, consts.DefaultAuthWindowLength),
		seed:   maphash.MakeSeed(),
		keys:   map[uint64]*authAttempts{},
	}
}

// acquire reserves a credential check for key, waiting while other checks for the same key are in
// flight. It only fails when the key already reached `limit` failures in the current window.
func (l *authLimiter) acquire(ctx context.Context, key string) (*authSlot, bool) {
	if l == nil {
		return nil, true
	}
	a, ok := l.reserve(key)
	if !ok {
		return nil, false
	}

	select {
	case a.slots <- struct{}{}:
	case <-ctx.Done():
		l.unref(a)
		return nil, false
	}

	l.mu.Lock()
	blocked := a.failures >= l.limit
	if blocked {
		a.refs--
	}
	l.mu.Unlock()
	if blocked {
		<-a.slots
		return nil, false
	}
	return &authSlot{limiter: l, entry: a}, true
}

func (l *authLimiter) reserve(key string) (*authAttempts, bool) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweep(now)

	h := maphash.String(l.seed, key)
	a := l.keys[h]
	switch {
	case a == nil:
		a = &authAttempts{start: now, slots: make(chan struct{}, l.limit)}
		l.keys[h] = a
	case now.Sub(a.start) >= l.window:
		a.failures, a.start = 0, now
	}
	if a.failures >= l.limit {
		return nil, false
	}
	a.refs++
	return a, true
}

func (l *authLimiter) unref(a *authAttempts) {
	l.mu.Lock()
	a.refs--
	l.mu.Unlock()
}

func (s *authSlot) release(failed bool) {
	if s == nil {
		return
	}
	s.limiter.mu.Lock()
	if failed {
		s.entry.failures++
	}
	s.entry.refs--
	s.limiter.mu.Unlock()
	<-s.entry.slots
}

// sweep drops idle expired keys once per window, so memory is bounded by recent attempts.
func (l *authLimiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < l.window {
		return
	}
	for h, a := range l.keys {
		if a.refs == 0 && now.Sub(a.start) >= l.window {
			delete(l.keys, h)
		}
	}
	l.lastSweep = now
}

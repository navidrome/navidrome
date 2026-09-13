package subsonic

import (
	"cmp"
	"hash/maphash"
	"sync"
	"time"

	"github.com/navidrome/navidrome/consts"
)

// authLimiter caps failed Subsonic logins per key. Callers acquire a slot before checking the
// credentials and release it on success, so concurrent guesses cannot overshoot the limit.
type authLimiter struct {
	limit     int
	window    time.Duration
	seed      maphash.Seed
	mu        sync.Mutex
	keys      map[uint64]*authAttempts
	lastSweep time.Time
}

type authAttempts struct {
	count int
	start time.Time
}

// newAuthLimiter returns nil when limit is not positive. A nil limiter allows everything.
func newAuthLimiter(limit int, window time.Duration) *authLimiter {
	if limit <= 0 {
		return nil
	}
	return &authLimiter{
		limit:     limit,
		window:    cmp.Or(window, consts.DefaultAuthWindowLength),
		seed:      maphash.MakeSeed(),
		keys:      map[uint64]*authAttempts{},
		lastSweep: time.Now(),
	}
}

func (l *authLimiter) acquire(key string) bool {
	if l == nil {
		return true
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweep(now)

	h := maphash.String(l.seed, key)
	a := l.keys[h]
	if a == nil || now.Sub(a.start) >= l.window {
		a = &authAttempts{start: now}
		l.keys[h] = a
	}
	if a.count >= l.limit {
		return false
	}
	a.count++
	return true
}

func (l *authLimiter) release(key string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if a := l.keys[maphash.String(l.seed, key)]; a != nil && a.count > 0 {
		a.count--
	}
}

// sweep drops expired keys once per window, so memory is bounded by recent attempts.
func (l *authLimiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < l.window {
		return
	}
	for h, a := range l.keys {
		if now.Sub(a.start) >= l.window {
			delete(l.keys, h)
		}
	}
	l.lastSweep = now
}

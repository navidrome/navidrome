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
	keys      map[uint64]authAttempts // hashed, so attacker-chosen usernames cannot bloat memory
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
		limit:  limit,
		window: cmp.Or(window, consts.DefaultAuthWindowLength),
		seed:   maphash.MakeSeed(),
		keys:   map[uint64]authAttempts{},
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
	if now.Sub(a.start) >= l.window {
		a = authAttempts{start: now}
	}
	if a.count >= l.limit {
		return false
	}
	a.count++
	l.keys[h] = a
	return true
}

func (l *authLimiter) release(key string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	h := maphash.String(l.seed, key)
	if a, ok := l.keys[h]; ok && a.count > 0 {
		a.count--
		l.keys[h] = a
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

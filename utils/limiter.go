package utils

import (
	"cmp"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter is a rate limiter that allows a function to be executed at most once per ID and per interval.
type Limiter struct {
	Interval time.Duration
	sm       sync.Map
}

// Do executes the provided function `f` if the rate limiter for the given `id` allows it.
// It uses the interval specified in the Limiter struct or defaults to 1 minute if not set.
func (m *Limiter) Do(id string, f func()) {
	interval := cmp.Or(
		m.Interval,
		time.Minute, // Default every 1 minute
	)
	limiter, _ := m.sm.LoadOrStore(id, &rate.Sometimes{Interval: interval})
	limiter.(*rate.Sometimes).Do(f)
}

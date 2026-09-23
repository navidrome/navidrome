package scanner

import "time"

func SetPersistRetryDelay(d time.Duration) func() {
	prev := persistRetryDelay
	persistRetryDelay = d
	return func() { persistRetryDelay = prev }
}

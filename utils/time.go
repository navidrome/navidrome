package utils

import "time"

// TimeNewest returns the most recent (latest) time from a list of provided times.
//
// Usage:
//
//	t1 := time.Date(2025, 8, 14, 10, 0, 0, 0, time.UTC)
//	t2 := time.Date(2025, 8, 14, 12, 0, 0, 0, time.UTC)
//	newest := TimeNewest(t1, t2) // returns t2
//
// If no times are provided, the function returns the zero value of time.Time.
// The zero value represents January 1, year 1, 00:00:00 UTC.
//
// Note:
//   - The function compares all times using the After method.
//   - Times with the same value are handled correctly; the first occurrence is returned.
func TimeNewest(times ...time.Time) time.Time {
	if len(times) == 0 {
		return time.Time{}
	}

	newest := times[0]
	for _, t := range times[1:] {
		if t.After(newest) {
			newest = t
		}
	}
	return newest
}

package utils

import "time"

func TimeNewest(times ...time.Time) time.Time {
	newest := time.Time{}
	for _, t := range times {
		if t.After(newest) {
			newest = t
		}
	}
	return newest
}

package utils

import "time"

func ToTime(millis int64) time.Time {
	return time.Unix(0, millis*int64(time.Millisecond))
}

func ToMillis(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

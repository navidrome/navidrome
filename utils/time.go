package utils

import "time"

func ToTime(millis int64) time.Time {
	t := time.Unix(0, millis*int64(time.Millisecond))
	return t.Local()
}

func ToMillis(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

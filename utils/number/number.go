package number

import (
	"strconv"
)

// Integer is a constraint that permits any integer type.
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func ParseInt[T Integer](s string) T {
	r, _ := strconv.ParseInt(s, 10, 64)
	return T(r)
}

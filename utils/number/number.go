package number

import (
	"strconv"

	"golang.org/x/exp/constraints"
)

func ParseInt[T constraints.Integer](s string) T {
	r, _ := strconv.ParseInt(s, 10, 64)
	return T(r)
}

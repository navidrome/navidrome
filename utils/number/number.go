package number

import (
	"crypto/rand"
	"math/big"

	"golang.org/x/exp/constraints"
)

// TODO Remove on Go 1.22, in favor of builtin `min` function.
func Min[T constraints.Ordered](vs ...T) T {
	if len(vs) == 0 {
		var zero T
		return zero
	}
	min := vs[0]
	for _, v := range vs[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

// TODO Remove on Go 1.22, in favor of builtin `max` function.
func Max[T constraints.Ordered](vs ...T) T {
	if len(vs) == 0 {
		var zero T
		return zero
	}
	max := vs[0]
	for _, v := range vs[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func RandomInt64(max int64) int64 {
	rnd, _ := rand.Int(rand.Reader, big.NewInt(max))
	return rnd.Int64()
}

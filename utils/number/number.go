package number

import (
	"crypto/rand"
	"math/big"
	"strconv"

	"golang.org/x/exp/constraints"
)

func RandomInt64(max int64) int64 {
	rnd, _ := rand.Int(rand.Reader, big.NewInt(max))
	return rnd.Int64()
}

func ParseInt[T constraints.Integer](s string) T {
	r, _ := strconv.ParseInt(s, 10, 64)
	return T(r)
}

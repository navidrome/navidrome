package random

import (
	"crypto/rand"
	"math/big"

	"golang.org/x/exp/constraints"
)

func Int64[T constraints.Integer](max T) int64 {
	rnd, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return rnd.Int64()
}

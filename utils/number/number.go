package number

import (
	"crypto/rand"
	"math/big"
)

func RandomInt64(max int64) int64 {
	rnd, _ := rand.Int(rand.Reader, big.NewInt(max))
	return rnd.Int64()
}

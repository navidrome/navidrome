package random

import (
	"crypto/rand"
	"encoding/binary"
	"math/big"

	"github.com/navidrome/navidrome/utils/number"
)

// Int64N returns a random int64 between 0 and max.
// This is a reimplementation of math/rand/v2.Int64N using a cryptographically secure random number generator.
func Int64N[T number.Integer](max T) int64 {
	rnd, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return rnd.Int64()
}

// Uint64 returns a random uint64.
// This is a reimplementation of math/rand/v2.Uint64 using a cryptographically secure random number generator.
func Uint64() uint64 {
	buffer := make([]byte, 8)
	_, _ = rand.Read(buffer)
	return binary.BigEndian.Uint64(buffer)
}

package migrations

import (
	"crypto/md5"
	"encoding/hex"
	"math/big"

	"github.com/navidrome/navidrome/model/id"
)

// canonicalID maps any historical Navidrome id shape to the canonical 22-char base62 encoding
// of a 128-bit value; unrecognized shapes (including empty and share ids) pass through unchanged.
func canonicalID(s string) string {
	switch len(s) {
	case 22:
		v, ok := new(big.Int).SetString(s, 62)
		if !ok || v.Sign() < 0 || v.BitLen() <= 128 {
			return s
		}
		sum := md5.Sum([]byte(s))
		return id.Encode128(sum)
	case 32:
		b, err := hex.DecodeString(s)
		if err != nil {
			return s
		}
		return id.Encode128([16]byte(b))
	case 36:
		if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
			return s
		}
		b, err := hex.DecodeString(s[:8] + s[9:13] + s[14:18] + s[19:23] + s[24:])
		if err != nil {
			return s
		}
		return id.Encode128([16]byte(b))
	}
	return s
}

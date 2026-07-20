package id

import (
	"crypto/md5"
	"fmt"
	"math/big"
	"strings"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/nanoid"
)

func NewRandom() string {
	id, err := nanoid.Generate("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", 22)
	if err != nil {
		log.Error("Could not generate new ID", err)
	}
	return id
}

// Encode128 renders a 16-byte value as the canonical 22-char zero-padded base62 id.
func Encode128(b []byte) string {
	return fmt.Sprintf("%022s", new(big.Int).SetBytes(b).Text(62))
}

// Decode128 is the exact inverse of Encode128.
func Decode128(s string) ([]byte, error) {
	if len(s) != 22 {
		return nil, fmt.Errorf("invalid id length %d", len(s))
	}
	v, ok := new(big.Int).SetString(s, 62)
	if !ok || v.Sign() < 0 {
		return nil, fmt.Errorf("invalid base62 id %q", s)
	}
	if v.BitLen() > 128 {
		return nil, fmt.Errorf("id %q overflows 128 bits", s)
	}
	return v.FillBytes(make([]byte, 16)), nil
}

func NewHash(data ...string) string {
	hash := md5.New()
	for _, d := range data {
		hash.Write([]byte(d))
		hash.Write([]byte(string('\u200b')))
	}
	return Encode128(hash.Sum(nil))
}

func NewTagID(name, value string) string {
	return NewHash(strings.ToLower(name), strings.ToLower(value))
}

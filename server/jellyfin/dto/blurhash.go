package dto

import (
	"fmt"
	"hash/fnv"
	"time"
)

// base83Alphabet is the blurhash spec's base83 encoding alphabet; order is part of the spec.
const base83Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz#$%*+,-.:;=?@[]^_{|}~"

// base83 encodes value as a fixed-width, big-endian base83 string of the given length.
func base83(value, length int) string {
	b := make([]byte, length)
	for i := 1; i <= length; i++ {
		digit := (value / pow83(length-i)) % 83
		b[i-1] = base83Alphabet[digit]
	}
	return string(b)
}

func pow83(n int) int {
	result := 1
	for range n {
		result *= 83
	}
	return result
}

// blurHash returns a valid 6-char blurhash for a solid color derived from seed. Finamp only needs a
// well-formed, per-tag-stable value (it uses this as a download de-dup key and blur placeholder), so
// a solid color unique to the tag satisfies both without decoding cover art.
func blurHash(seed string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	sum := h.Sum(nil)
	r, g, b := int(sum[0]), int(sum[1]), int(sum[2])
	dc := (r << 16) | (g << 8) | b
	return "00" + base83(dc, 4)
}

// primaryBlurHash returns the stored blurhash when it was computed from the entity's current
// artwork version or later (the snapshot folds in image file mtimes, which can exceed row
// timestamps); otherwise a fake seeded by id+version, so the value still rotates on any artwork
// change (Finamp keys its cover caches by this value; tags never reach its image URLs).
func primaryBlurHash(stored string, storedAt *time.Time, id string, version time.Time) string {
	if stored != "" && storedAt != nil && !storedAt.Before(version) {
		return stored
	}
	return blurHash(fmt.Sprintf("%s-%x", id, version.UnixMilli()))
}

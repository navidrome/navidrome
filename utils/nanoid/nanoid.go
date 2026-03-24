package nanoid

import (
	"crypto/rand"
	"errors"
	"math"
)

// Generate returns a cryptographically secure random string of `size` characters
// drawn from `alphabet`. It uses bitmask with rejection sampling to avoid modulo bias.
// The alphabet must be non-empty, contain at most 255 characters, and consist only of
// ASCII characters. Non-ASCII alphabets (e.g., multi-byte UTF-8) are not supported.
func Generate(alphabet string, size int) (string, error) {
	if len(alphabet) == 0 || len(alphabet) > 255 {
		return "", errors.New("alphabet must be non-empty and at most 255 characters")
	}
	if size <= 0 {
		return "", errors.New("size must be a positive integer")
	}

	mask := getMask(len(alphabet))
	step := int(math.Ceil(1.6 * float64(mask) * float64(size) / float64(len(alphabet))))

	id := make([]byte, size)
	bytes := make([]byte, step)
	for j := 0; ; {
		if _, err := rand.Read(bytes); err != nil {
			return "", err
		}
		for i := range step {
			idx := int(bytes[i]) & mask
			if idx < len(alphabet) {
				id[j] = alphabet[idx]
				j++
				if j == size {
					return string(id), nil
				}
			}
		}
	}
}

// getMask returns the smallest bitmask >= alphabetSize-1.
func getMask(alphabetSize int) int {
	for i := 1; i <= 8; i++ {
		mask := (2 << uint(i)) - 1
		if mask >= alphabetSize-1 {
			return mask
		}
	}
	return 0
}

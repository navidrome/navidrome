package str

import (
	"strings"
)

// NaturalSortCompare compares two strings using natural sort ordering,
// where embedded numeric sequences are compared as numbers rather than
// lexicographically. For example, "track2" < "track10" (unlike lexicographic
// ordering which gives "track10" < "track2"). The comparison is also
// case-insensitive.
func NaturalSortCompare(a, b string) int {
	ia, ib := 0, 0
	for ia < len(a) && ib < len(b) {
		ca := a[ia]
		cb := b[ib]

		// If both characters are digits, compare the full numeric sequences
		if isDigit(ca) && isDigit(cb) {
			result := compareNumericChunks(a, b, &ia, &ib)
			if result != 0 {
				return result
			}
			continue
		}

		// Case-insensitive character comparison
		la := toLower(ca)
		lb := toLower(cb)
		if la != lb {
			if la < lb {
				return -1
			}
			return 1
		}

		ia++
		ib++
	}

	// The shorter string comes first if all else is equal
	return len(a) - len(b)
}

// compareNumericChunks compares two numeric sequences starting at positions
// ia and ib in strings a and b. It advances the position indices past the
// numeric sequences. Numbers are compared by value, with leading zeros
// used as a tiebreaker (fewer leading zeros comes first).
func compareNumericChunks(a, b string, ia, ib *int) int {
	// Skip leading zeros and count them
	zerosA := 0
	for *ia < len(a) && a[*ia] == '0' {
		zerosA++
		*ia++
	}
	zerosB := 0
	for *ib < len(b) && b[*ib] == '0' {
		zerosB++
		*ib++
	}

	// Find the extent of the remaining digits
	startA := *ia
	for *ia < len(a) && isDigit(a[*ia]) {
		*ia++
	}
	startB := *ib
	for *ib < len(b) && isDigit(b[*ib]) {
		*ib++
	}

	lenA := *ia - startA
	lenB := *ib - startB

	// More significant digits means a larger number
	if lenA != lenB {
		return lenA - lenB
	}

	// Same number of significant digits - compare digit by digit
	for i := 0; i < lenA; i++ {
		if a[startA+i] != b[startB+i] {
			if a[startA+i] < b[startB+i] {
				return -1
			}
			return 1
		}
	}

	// Same numeric value - fewer leading zeros comes first
	if zerosA != zerosB {
		return zerosA - zerosB
	}

	return 0
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}

// NaturalSortKey transforms a string into a key that, when compared
// lexicographically with NOCASE collation, produces natural sort order.
// Numeric sequences are zero-padded to a fixed width so that lexicographic
// comparison yields numeric ordering.
func NaturalSortKey(s string) string {
	const padWidth = 20 // Enough for uint64 max

	var b strings.Builder
	b.Grow(len(s) + padWidth) // Pre-allocate a reasonable size

	i := 0
	for i < len(s) {
		if isDigit(s[i]) {
			// Find the full numeric sequence
			start := i
			for i < len(s) && isDigit(s[i]) {
				i++
			}
			numStr := s[start:i]

			// Pad the number with leading zeros to padWidth
			if len(numStr) < padWidth {
				for j := 0; j < padWidth-len(numStr); j++ {
					b.WriteByte('0')
				}
			}
			b.WriteString(numStr)
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

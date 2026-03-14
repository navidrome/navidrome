// Package natural provides natural (alphanumeric) string comparison.
// When both strings have digit sequences at the same position, they are
// compared numerically (so "file2" < "file10"); otherwise bytes are
// compared one-by-one. No allocations are made.
package natural

import "strings"

// Compare returns a negative value if a < b, zero if a == b,
// or a positive value if a > b using natural sort ordering.
//
// When two numeric segments are numerically equal (e.g. "01" vs "1"),
// comparison continues with the remaining suffixes. If one or both
// strings end at the digit boundary, the raw strings are compared
// lexically, which makes leading zeros significant as a tie-breaker
// (e.g. "a01" < "a1", "a0" < "a00").
func Compare(a, b string) int {
	ia, ib := 0, 0
	for ia < len(a) && ib < len(b) {
		ca, cb := a[ia], b[ib]
		da, db := isDigit(ca), isDigit(cb)

		switch {
		case da && db:
			// Both are in digit sequences — compare numerically.
			endA := ia
			for endA < len(a) && isDigit(a[endA]) {
				endA++
			}
			endB := ib
			for endB < len(b) && isDigit(b[endB]) {
				endB++
			}

			if c := compareNumbers(a[ia:endA], b[ib:endB]); c != 0 {
				return c
			}

			// Numerically equal. If both sides have trailing data, continue
			// comparing after the digit runs. Otherwise fall through to
			// lexical comparison of the full remaining strings (which makes
			// leading-zero differences significant as a tie-breaker).
			if endA < len(a) && endB < len(b) {
				ia = endA
				ib = endB
				continue
			}
			return strings.Compare(a[ia:], b[ib:])
		case da != db:
			return int(ca) - int(cb)
		default:
			if ca != cb {
				return int(ca) - int(cb)
			}
			ia++
			ib++
		}
	}
	return (len(a) - ia) - (len(b) - ib)
}

// compareNumbers compares two digit strings numerically.
// Leading zeros are stripped before comparison.
func compareNumbers(a, b string) int {
	// Strip leading zeros.
	sa := stripZeros(a)
	sb := stripZeros(b)

	// Different lengths after stripping means different magnitude.
	if len(sa) != len(sb) {
		return len(sa) - len(sb)
	}

	// Same length — compare digit by digit.
	for i := range len(sa) {
		if sa[i] != sb[i] {
			return int(sa[i]) - int(sb[i])
		}
	}
	return 0
}

// stripZeros returns s with leading '0' bytes removed.
// If s is all zeros, returns the last byte (a single "0").
func stripZeros(s string) string {
	i := 0
	for i < len(s) && s[i] == '0' {
		i++
	}
	if i == len(s) && len(s) > 0 {
		return s[len(s)-1:]
	}
	return s[i:]
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

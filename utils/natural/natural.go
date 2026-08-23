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
// comparison continues with the remaining suffixes, and the padding
// difference is kept as a final tie-breaker that only decides strings
// that are otherwise equal (e.g. "a01" < "a1", "a0" < "a00"). Deferring
// it that way is what keeps the ordering transitive, which SQLite
// requires of a collating function.
func Compare(a, b string) int {
	return compare(a, b, false)
}

// CompareFold is Compare with ASCII case folding, matching SQLite's NOCASE
// collation: only A-Z fold, bytes >= 0x80 are compared as-is.
func CompareFold(a, b string) int {
	return compare(a, b, true)
}

func compare(a, b string, fold bool) int {
	ia, ib := 0, 0
	// Set when two runs are numerically equal but differently padded. Applying it
	// immediately would break transitivity, so it only decides otherwise-equal strings.
	padTie := 0
	for ia < len(a) && ib < len(b) {
		ca, cb := a[ia], b[ib]
		da, db := isDigit(ca), isDigit(cb)
		if fold {
			ca, cb = lower(ca), lower(cb)
		}

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
			if t := strings.Compare(a[ia:endA], b[ib:endB]); t != 0 {
				padTie = t
			}
			ia = endA
			ib = endB
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
	if c := (len(a) - ia) - (len(b) - ib); c != 0 {
		return c
	}
	return padTie
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

func lower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}

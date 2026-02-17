package str

import (
	"strings"

	"github.com/maruel/natural"
)

// NaturalSortCompare compares two strings using natural sort ordering,
// where embedded numeric sequences are compared as numbers rather than
// lexicographically. For example, "track2" < "track10" (unlike lexicographic
// ordering which gives "track10" < "track2"). The comparison is also
// case-insensitive.
func NaturalSortCompare(a, b string) int {
	return natural.Compare(strings.ToLower(a), strings.ToLower(b))
}

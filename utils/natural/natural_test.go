package natural_test

import (
	"testing"

	"github.com/navidrome/navidrome/utils/natural"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestNatural(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Natural Suite")
}

// expectOrder asserts the sign of cmp(a, b) matches expected.
func expectOrder(cmp func(string, string) int, a, b string, expected int) {
	result := cmp(a, b)
	switch {
	case expected < 0:
		ExpectWithOffset(1, result).To(BeNumerically("<", 0), "expected %q < %q", a, b)
	case expected > 0:
		ExpectWithOffset(1, result).To(BeNumerically(">", 0), "expected %q > %q", a, b)
	default:
		ExpectWithOffset(1, result).To(Equal(0), "expected %q == %q", a, b)
	}
}

var _ = Describe("Compare", func() {
	DescribeTable("returns correct ordering",
		func(a, b string, expected int) {
			expectOrder(natural.Compare, a, b, expected)
		},
		// Basic string ordering
		Entry("a < b", "a", "b", -1),
		Entry("b > a", "b", "a", 1),
		Entry("a < aa (prefix)", "a", "aa", -1),
		Entry("aa > a", "aa", "a", 1),

		// Equal strings
		Entry("equal strings return 0", "abc", "abc", 0),
		Entry("both empty", "", "", 0),
		Entry("a01 == a01", "a01", "a01", 0),
		Entry("a1 == a1", "a1", "a1", 0),

		// Empty string edge cases
		Entry("empty < non-empty", "", "a", -1),
		Entry("non-empty > empty", "a", "", 1),

		// Numeric comparison
		Entry("2 < 10 numerically", "2", "10", -1),
		Entry("10 > 2 numerically", "10", "2", 1),
		Entry("equal numbers", "42", "42", 0),
		Entry("9 < 10", "9", "10", -1),
		Entry("99 < 100", "99", "100", -1),

		// Simple numeric segments (from original library)
		Entry("a0 < a1", "a0", "a1", -1),
		Entry("a0 < a00", "a0", "a00", -1),
		Entry("a00 < a01", "a00", "a01", -1),
		Entry("a01 < a1", "a01", "a1", -1),
		Entry("a01 < a2", "a01", "a2", -1),
		Entry("a01x < a2x", "a01x", "a2x", -1),
		Entry("a01 > a00", "a01", "a00", 1),
		Entry("a2 > a01", "a2", "a01", 1),
		Entry("a2x > a01x", "a2x", "a01x", 1),

		// Multiple numeric groups (from original library)
		Entry("a0b00 < a00b1", "a0b00", "a00b1", -1),
		Entry("a0b00 < a00b01", "a0b00", "a00b01", -1),
		Entry("a00b0 < a0b00", "a00b0", "a0b00", -1),
		Entry("a00b00 < a0b01", "a00b00", "a0b01", -1),
		Entry("a00b00 < a0b1", "a00b00", "a0b1", -1),
		Entry("a00b00 > a0b0", "a00b00", "a0b0", 1),
		Entry("a00b01 > a0b00", "a00b01", "a0b00", 1),
		// Distinct strings must not compare equal: the padding difference in the first
		// run decides once everything else matches.
		Entry("a00b00 > a0b00", "a00b00", "a0b00", 1),

		// Leading zeros at end of string — lexical tie-break
		Entry("file01 < file1", "file01", "file1", -1),

		// Prefix comparison
		Entry("abc < abcd", "abc", "abcd", -1),
		Entry("abcd > abc", "abcd", "abc", 1),

		// Navidrome use cases: cover art sorting
		Entry("cover < cover.1", "cover", "cover.1", -1),
		Entry("cover.1 < cover.2", "cover.1", "cover.2", -1),
		Entry("cover.2 < cover.10", "cover.2", "cover.10", -1),

		// Navidrome use cases: disc sorting
		Entry("disc1 < disc2", "disc1", "disc2", -1),
		Entry("disc2 < disc10", "disc2", "disc10", -1),
		Entry("disc1 < disc10", "disc1", "disc10", -1),

		// Multiple numeric segments
		Entry("a1b2 < a1b10", "a1b2", "a1b10", -1),
		Entry("a2b1 > a1b2", "a2b1", "a1b2", 1),

		// Numbers at the start
		Entry("2abc < 10abc", "2abc", "10abc", -1),

		// Numbers larger than uint64 max (from original library)
		Entry("large: fewer digits < more digits",
			"a99999999999999999999", "a100000000000000000000", -1),
		Entry("large: digit-by-digit comparison",
			"a123456789012345678901234567890", "a123456789012345678901234567891", -1),
		Entry("large: more digits > fewer digits",
			"a999999999999999999999", "a1000000000000000000000", -1),
		Entry("large: 20 digits < 100 digits by length",
			"a20000000000000000000", "a100000000000000000000", -1),
		Entry("large: 100 digits > 20 digits",
			"a100000000000000000000", "a20000000000000000000", 1),
		Entry("large: reverse of above",
			"a1000000000000000000000", "a999999999999999999999", 1),
		Entry("large: equal",
			"a100000000000000000000", "a100000000000000000000", 0),
		Entry("large: leading zeros with trailing data",
			"a00000000000000000000001x", "a1x", -1),
		Entry("large: leading zeros with trailing data (2)",
			"a099999999999999999999x", "a99999999999999999999x", -1),
	)
})

var _ = Describe("CompareFold", func() {
	DescribeTable("orders case-insensitively",
		func(a, b string, expected int) {
			expectOrder(natural.CompareFold, a, b, expected)
		},
		Entry("numbers compare numerically", "foo 2", "foo 10", -1),
		Entry("numbers compare numerically, reversed", "foo 10", "foo 2", 1),
		Entry("case is ignored", "apple 2", "Banana 10", -1),
		Entry("case is ignored, reversed", "Banana 10", "apple 2", 1),
		Entry("same word, different case, is equal", "ABC", "abc", 0),
		Entry("case ignored while comparing numbers", "Vol 2", "vol 10", -1),
		Entry("uppercase digits boundary", "Track9", "track10", -1),
		Entry("empty vs empty", "", "", 0),
		Entry("empty sorts first", "", "a", -1),
		Entry("non-ASCII is left untouched", "café 2", "café 10", -1),
	)

	// SQLite requires a collating function to be transitive; if it is not, the behavior of
	// ORDER BY is undefined and paginated queries can drop or duplicate rows.
	It("is transitive, as a SQLite collation requires", func() {
		var corpus []string
		var build func(prefix string, depth int)
		build = func(prefix string, depth int) {
			if prefix != "" {
				corpus = append(corpus, prefix)
			}
			if depth == 0 {
				return
			}
			for _, c := range []string{"0", "1", "a"} {
				build(prefix+c, depth-1)
			}
		}
		build("", 3)

		sign := func(n int) int {
			switch {
			case n < 0:
				return -1
			case n > 0:
				return 1
			}
			return 0
		}
		for _, a := range corpus {
			for _, b := range corpus {
				ab := sign(natural.CompareFold(a, b))
				for _, c := range corpus {
					bc := sign(natural.CompareFold(b, c))
					ac := sign(natural.CompareFold(a, c))
					if ab == 0 && bc == 0 {
						Expect(ac).To(Equal(0), "%q==%q and %q==%q but %q vs %q is %d", a, b, b, c, a, c, ac)
					}
					if ab < 0 && bc < 0 {
						Expect(ac).To(BeNumerically("<", 0), "%q<%q<%q but %q vs %q is %d", a, b, c, a, c, ac)
					}
				}
			}
		}
	})

	It("matches Compare when both sides are already lowercase", func() {
		pairs := [][2]string{{"foo 2", "foo 10"}, {"a01", "a1"}, {"a", "aa"}, {"vol 3", "vol 3"}}
		for _, p := range pairs {
			Expect(natural.CompareFold(p[0], p[1])).To(Equal(natural.Compare(p[0], p[1])),
				"CompareFold(%q,%q) should match Compare", p[0], p[1])
		}
	})
})

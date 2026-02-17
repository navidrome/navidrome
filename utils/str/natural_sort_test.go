package str_test

import (
	"testing"

	"github.com/navidrome/navidrome/utils/str"
)

func TestNaturalSortCompare(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int // -1, 0, or 1
	}{
		// Basic string comparison
		{"equal strings", "abc", "abc", 0},
		{"less than", "abc", "abd", -1},
		{"greater than", "abd", "abc", 1},

		// Case insensitive
		{"case insensitive equal", "ABC", "abc", 0},
		{"case insensitive less", "ABC", "abd", -1},
		{"case insensitive mixed", "aBc", "AbC", 0},

		// Numeric ordering
		{"single digit order", "track1", "track2", -1},
		{"natural sort 2 vs 10", "track2", "track10", -1},
		{"natural sort 9 vs 10", "track9", "track10", -1},
		{"natural sort 10 vs 2", "track10", "track2", 1},
		{"natural sort 1 vs 100", "track1", "track100", -1},

		// Album numbering (main use case from issue)
		{"album numbers 1 vs 2", "Bravo Hits 1", "Bravo Hits 2", -1},
		{"album numbers 2 vs 10", "Bravo Hits 2", "Bravo Hits 10", -1},
		{"album numbers 10 vs 100", "Bravo Hits 10", "Bravo Hits 100", -1},
		{"album numbers 9 vs 10", "Bravo Hits 9", "Bravo Hits 10", -1},
		{"album numbers equal", "Bravo Hits 10", "Bravo Hits 10", 0},
		{"album numbers 99 vs 100", "Bravo Hits 99", "Bravo Hits 100", -1},
		{"album numbers 100 vs 101", "Bravo Hits 100", "Bravo Hits 101", -1},

		// Pure numeric strings
		{"pure number 1 vs 2", "1", "2", -1},
		{"pure number 2 vs 10", "2", "10", -1},
		{"pure number 10 vs 9", "10", "9", 1},
		{"pure number equal", "42", "42", 0},

		// Leading zeros (same numeric value, but more leading zeros sorts later)
		{"leading zeros same value", "01", "1", 1},
		{"leading zeros 02 vs 1", "02", "1", 1},
		{"leading zeros 01 vs 10", "01", "10", -1},

		// Multiple numeric sequences
		{"multi number 1.2 vs 1.10", "file1.2", "file1.10", -1},
		{"multi number 2.1 vs 10.1", "file2.1", "file10.1", -1},

		// Empty strings
		{"both empty", "", "", 0},
		{"empty vs non-empty", "", "a", -1},
		{"non-empty vs empty", "a", "", 1},

		// Strings with only numbers at different positions
		{"number prefix vs alpha prefix", "1abc", "abc", -1},
		{"alpha prefix vs number prefix", "abc", "1abc", 1},

		// Edge cases
		{"same prefix different length", "abc", "abcd", -1},
		{"numbers at beginning", "10abc", "9abc", 1},
		{"numbers at end", "abc10", "abc9", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := str.NaturalSortCompare(tt.a, tt.b)
			// Normalize to -1, 0, 1 for comparison
			gotNorm := normalize(got)
			if gotNorm != tt.want {
				t.Errorf("NaturalSortCompare(%q, %q) = %d (normalized: %d), want %d", tt.a, tt.b, got, gotNorm, tt.want)
			}
		})
	}
}

func TestNaturalSortCompare_Symmetry(t *testing.T) {
	pairs := [][2]string{
		{"track2", "track10"},
		{"abc", "abd"},
		{"Bravo Hits 9", "Bravo Hits 10"},
		{"1", "2"},
		{"file1.2", "file1.10"},
	}

	for _, pair := range pairs {
		a, b := pair[0], pair[1]
		ab := str.NaturalSortCompare(a, b)
		ba := str.NaturalSortCompare(b, a)
		if normalize(ab) != -normalize(ba) {
			t.Errorf("Symmetry violated: Compare(%q,%q)=%d but Compare(%q,%q)=%d", a, b, ab, b, a, ba)
		}
	}
}

func TestNaturalSortKey(t *testing.T) {
	// Test that sorting by NaturalSortKey produces natural order
	inputs := []string{
		"Bravo Hits 1",
		"Bravo Hits 10",
		"Bravo Hits 100",
		"Bravo Hits 2",
		"Bravo Hits 20",
		"Bravo Hits 3",
		"Bravo Hits 9",
	}
	expected := []string{
		"Bravo Hits 1",
		"Bravo Hits 2",
		"Bravo Hits 3",
		"Bravo Hits 9",
		"Bravo Hits 10",
		"Bravo Hits 20",
		"Bravo Hits 100",
	}

	// Get the keys and verify they sort correctly
	keys := make([]string, len(inputs))
	for i, s := range inputs {
		keys[i] = str.NaturalSortKey(s)
	}

	// Verify the expected order produces keys in ascending order
	expectedKeys := make([]string, len(expected))
	for i, s := range expected {
		expectedKeys[i] = str.NaturalSortKey(s)
	}
	for i := 1; i < len(expectedKeys); i++ {
		if expectedKeys[i] <= expectedKeys[i-1] {
			t.Errorf("NaturalSortKey order violation: key(%q) = %q should be > key(%q) = %q",
				expected[i], expectedKeys[i], expected[i-1], expectedKeys[i-1])
		}
	}
}

func normalize(v int) int {
	if v < 0 {
		return -1
	}
	if v > 0 {
		return 1
	}
	return 0
}

package artwork

import (
	"path/filepath"
	"strconv"
	"strings"
)

// extractDiscNumber extracts a disc number from a filename based on a glob pattern.
// It finds the portion of the filename that the wildcard matched and parses leading
// digits as the disc number. Returns (0, false) if the pattern doesn't match or
// no leading digits are found in the wildcard portion.
func extractDiscNumber(pattern, filename string) (int, bool) {
	filename = strings.ToLower(filename)
	pattern = strings.ToLower(pattern)

	matched, err := filepath.Match(pattern, filename)
	if err != nil || !matched {
		return 0, false
	}

	// Find the prefix before the first '*' in the pattern
	starIdx := strings.IndexByte(pattern, '*')
	if starIdx < 0 {
		return 0, false
	}
	prefix := pattern[:starIdx]

	// Strip the prefix from the filename to get the wildcard-matched portion
	if !strings.HasPrefix(filename, prefix) {
		return 0, false
	}
	remainder := filename[len(prefix):]

	// Extract leading ASCII digits from the remainder
	var digits []byte
	for _, r := range remainder {
		if r >= '0' && r <= '9' {
			digits = append(digits, byte(r))
		} else {
			break
		}
	}

	if len(digits) == 0 {
		return 0, false
	}

	num, err := strconv.Atoi(string(digits))
	if err != nil {
		return 0, false
	}
	return num, true
}

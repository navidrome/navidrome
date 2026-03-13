package artwork

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/log"
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

// fromDiscExternalFile returns a sourceFunc that matches image files against a glob
// pattern with disc-number-aware filtering.
//
// Matching rules:
//   - If a disc number can be extracted from the filename, the file matches only if
//     the number equals the target disc number.
//   - If no number is found and this is a multi-folder album, the file matches if
//     it's in a folder containing tracks for this disc.
//   - If no number is found and this is a single-folder album, the file is skipped
//     (ambiguous).
func fromDiscExternalFile(ctx context.Context, files []string, pattern string, discNumber int, discFolders map[string]bool, isMultiFolder bool) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		for _, file := range files {
			_, name := filepath.Split(file)
			match, err := filepath.Match(pattern, strings.ToLower(name))
			if err != nil {
				log.Warn(ctx, "Error matching disc art file to pattern", "pattern", pattern, "file", file)
				continue
			}
			if !match {
				continue
			}

			// Try to extract disc number from filename
			num, hasNum := extractDiscNumber(pattern, name)
			if hasNum {
				// File has a disc number — must match target disc
				if num != discNumber {
					continue
				}
			} else if isMultiFolder {
				// No number, multi-folder: match by folder association
				dir := filepath.Dir(file)
				if !discFolders[dir] {
					continue
				}
			} else {
				// No number, single-folder: ambiguous, skip
				continue
			}

			f, err := os.Open(file)
			if err != nil {
				log.Warn(ctx, "Could not open disc art file", "file", file, err)
				continue
			}
			return f, file, nil
		}
		return nil, "", fmt.Errorf("disc %d: pattern '%s' not matched by files", discNumber, pattern)
	}
}

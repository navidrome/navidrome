package str

import (
	"strings"
	"unicode/utf8"
)

var utf8ToAscii = func() *strings.Replacer {
	var utf8Map = map[string]string{
		"'": `‘’‛′`,
		`"`: `＂〃ˮײ᳓″‶˶ʺ“”˝‟`,
		"-": `‐–—−―`,
	}

	list := make([]string, 0, len(utf8Map)*2)
	for ascii, utf8 := range utf8Map {
		for _, r := range utf8 {
			list = append(list, string(r), ascii)
		}
	}
	return strings.NewReplacer(list...)
}()

func Clear(name string) string {
	return utf8ToAscii.Replace(name)
}

func LongestCommonPrefix(list []string) string {
	if len(list) == 0 {
		return ""
	}

	for l := 0; l < len(list[0]); l++ {
		c := list[0][l]
		for i := 1; i < len(list); i++ {
			if l >= len(list[i]) || list[i][l] != c {
				return list[i][0:l]
			}
		}
	}
	return list[0]
}

// TruncateRunes truncates a string to a maximum number of runes, adding a suffix if truncated.
// The suffix is included in the rune count, so if maxRunes is 30 and suffix is "...", the actual
// string content will be truncated to fit within the maxRunes limit including the suffix.
func TruncateRunes(s string, maxRunes int, suffix string) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}

	suffixRunes := utf8.RuneCountInString(suffix)
	truncateAt := maxRunes - suffixRunes
	if truncateAt < 0 {
		truncateAt = 0
	}

	runes := []rune(s)
	if truncateAt >= len(runes) {
		return s + suffix
	}

	return string(runes[:truncateAt]) + suffix
}

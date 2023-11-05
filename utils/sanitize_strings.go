package utils

import (
	"html"
	"regexp"
	"strings"

	"github.com/deluan/sanitize"
	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/exp/slices"
)

var quotesRegex = regexp.MustCompile("[“”‘’'·\"\\[\\(\\{\\]\\)\\}]")
var singlequotesRegex = regexp.MustCompile("[‘’‛′]")
var doublequotesRegex = regexp.MustCompile("[＂〃ˮײ᳓″‶˶ʺ“”˝‟]")
var hyphenRegex = regexp.MustCompile("[‐–—−―]")

func SanitizeStrings(text ...string) string {
	sanitizedText := strings.Builder{}
	for _, txt := range text {
		sanitizedText.WriteString(strings.TrimSpace(sanitize.Accents(strings.ToLower(txt))) + " ")
	}
	words := make(map[string]struct{})
	for _, w := range strings.Fields(sanitizedText.String()) {
		words[w] = struct{}{}
	}
	var fullText []string
	for w := range words {
		w = quotesRegex.ReplaceAllString(w, "")
		w = singlequotesRegex.ReplaceAllString(w, "")
		w = doublequotesRegex.ReplaceAllString(w, "")
		w = hyphenRegex.ReplaceAllString(w, "-")
		if w != "" {
			fullText = append(fullText, w)
		}
	}
	slices.Sort(fullText)
	slices.Compact(fullText)
	return strings.Join(fullText, " ")
}

var policy = bluemonday.UGCPolicy()

func SanitizeText(text string) string {
	s := policy.Sanitize(text)
	return html.UnescapeString(s)
}

// replaces a set of problematic Unicode characters with their ascii equivalents
// maybe better implemented upstream in sanitize.Accents()
func SanitizeProblematicChars(text []string) []string {
	var sanitizedText []string
	for _, w := range text {
		w = singlequotesRegex.ReplaceAllString(w, "'")
		w = doublequotesRegex.ReplaceAllString(w, "\"")
		w = hyphenRegex.ReplaceAllString(w, "-")
		sanitizedText = append(sanitizedText, w)
	}
	return sanitizedText
}

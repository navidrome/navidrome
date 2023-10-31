package utils

import (
	"html"
	"regexp"
	"sort"
	"strings"

	"github.com/deluan/sanitize"
	"github.com/microcosm-cc/bluemonday"
	"github.com/navidrome/navidrome/utils/slice"
)

var hyphenRegex = regexp.MustCompile("[‐–—−―]")
var singlequotesRegex = regexp.MustCompile("[‘’‛′]")
var doublequotesRegex = regexp.MustCompile("[＂〃ˮײ᳓″‶˶ʺ“”˝‟]")

func SanitizeChars(text []string) []string {
	var fullText []string
	for _, w := range text {
		w = hyphenRegex.ReplaceAllString(w, "-")
		w = singlequotesRegex.ReplaceAllString(w, "'")
		w = doublequotesRegex.ReplaceAllString(w, "\"")
		fullText = append(fullText, w)
	}
	return fullText
}

var quotesRegex = regexp.MustCompile("[“”‘’'·\"\\[\\(\\{\\]\\)\\}]")

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
		w = hyphenRegex.ReplaceAllString(w, "-")
		if w != "" {
			fullText = append(fullText, w)
		}
	}
	concatenatedString := strings.Join(fullText, " ")
	allWords := slice.RemoveDuplicateStr(strings.Split(concatenatedString, " "))
	sort.Strings(allWords)
	return strings.Join(allWords, " ")
}

var policy = bluemonday.UGCPolicy()

func SanitizeText(text string) string {
	s := policy.Sanitize(text)
	return html.UnescapeString(s)
}

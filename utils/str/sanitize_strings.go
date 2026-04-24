package str

import (
	"html"
	"regexp"
	"slices"
	"strings"

	"github.com/deluan/sanitize"
	"github.com/microcosm-cc/bluemonday"
	"github.com/navidrome/navidrome/conf"
)

var ignoredCharsRegex = regexp.MustCompile("[“”‘’'\"\\[({\\])},]")
var slashRemover = strings.NewReplacer("\\", " ", "/", " ")

func SanitizeStrings(text ...string) string {
	// Concatenate all strings, removing extra spaces
	sanitizedText := strings.Builder{}
	for _, txt := range text {
		sanitizedText.WriteString(strings.TrimSpace(txt))
		sanitizedText.WriteByte(' ')
	}

	// Remove special symbols, accents, extra spaces and slashes
	sanitizedStrings := slashRemover.Replace(Clear(sanitizedText.String()))
	sanitizedStrings = sanitize.Accents(strings.ToLower(sanitizedStrings))
	sanitizedStrings = ignoredCharsRegex.ReplaceAllString(sanitizedStrings, "")
	fullText := strings.Fields(sanitizedStrings)

	// Remove duplicated words
	slices.Sort(fullText)
	fullText = slices.Compact(fullText)

	// Returns the sanitized text as a single string
	return strings.Join(fullText, " ")
}

var policy = bluemonday.UGCPolicy()

// SanitizeText unescapes the input string before sanitizing it as text.
// This should be used for fields rendered as plain text in the UI (e.g. lyrics, song titles, artist names)
func SanitizeText(text string) string {
	s := policy.Sanitize(text)
	return html.UnescapeString(s)
}

// SanitizeHTML unescapes the input string before sanitizing it as HTML.
// This should be used for fields rendered as HTML by clients (e.g. biographies, welcome messages)
// to prevent XSS bypasses via entity-encoded tags.
func SanitizeHTML(text string) string {
	return policy.Sanitize(html.UnescapeString(text))
}

func SanitizeFieldForSorting(originalValue string) string {
	v := strings.TrimSpace(sanitize.Accents(originalValue))
	return Clear(strings.ToLower(v))
}

func SanitizeFieldForSortingNoArticle(originalValue string) string {
	v := strings.TrimSpace(sanitize.Accents(originalValue))
	return Clear(strings.ToLower(strings.TrimSpace(RemoveArticle(v))))
}

func RemoveArticle(name string) string {
	articles := strings.SplitSeq(conf.Server.IgnoredArticles, " ")
	for a := range articles {
		n := strings.TrimPrefix(name, a+" ")
		if n != name {
			return n
		}
	}
	return name
}

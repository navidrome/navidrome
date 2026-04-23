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

func SanitizeText(text string) string {
	// Unescape HTML entities first so that payloads like
	// &lt;script&gt;alert(1)&lt;/script&gt; are fed to the sanitizer as
	// real tags it can strip. The previous order (sanitize then
	// unescape) let entity-encoded markup round-trip through the
	// bluemonday policy unchanged and then get decoded back to
	// dangerous HTML by html.UnescapeString — e.g. the Login.jsx
	// welcomeMessage rendering via dangerouslySetInnerHTML.
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

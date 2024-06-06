package str

import (
	"html"
	"regexp"
	"sort"
	"strings"

	"github.com/deluan/sanitize"
	"github.com/microcosm-cc/bluemonday"
	"github.com/navidrome/navidrome/conf"
)

var quotesRegex = regexp.MustCompile("[“”‘’'\"\\[\\(\\{\\]\\)\\}]")

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
		if w != "" {
			fullText = append(fullText, w)
		}
	}
	sort.Strings(fullText)
	return Clear(strings.Join(fullText, " "))
}

var policy = bluemonday.UGCPolicy()

func SanitizeText(text string) string {
	s := policy.Sanitize(text)
	return html.UnescapeString(s)
}

func SanitizeFieldForSorting(originalValue string) string {
	v := strings.TrimSpace(sanitize.Accents(originalValue))
	return Clear(strings.ToLower(v))
}

func SanitizeFieldForSortingNoArticle(originalValue string) string {
	v := strings.TrimSpace(sanitize.Accents(originalValue))
	return Clear(strings.ToLower(RemoveArticle(v)))
}

func RemoveArticle(name string) string {
	articles := strings.Split(conf.Server.IgnoredArticles, " ")
	for _, a := range articles {
		n := strings.TrimPrefix(name, a+" ")
		if n != name {
			return n
		}
	}
	return name
}

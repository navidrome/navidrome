package persistence

import (
	"regexp"
	"sort"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/kennygrant/sanitize"
)

var quotesRegex = regexp.MustCompile("[“”‘’'\"]")

func getFullText(text ...string) string {
	fullText := sanitizeStrings(text...)
	return " " + fullText
}

func sanitizeStrings(text ...string) string {
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
	return strings.Join(fullText, " ")
}

func (r sqlRepository) doSearch(q string, offset, size int, results interface{}, orderBys ...string) error {
	q = strings.TrimSuffix(q, "*")
	q = sanitizeStrings(q)
	if len(q) < 2 {
		return nil
	}
	sq := r.newSelectWithAnnotation(r.tableName + ".id").Columns("*")
	sq = sq.Limit(uint64(size)).Offset(uint64(offset))
	if len(orderBys) > 0 {
		sq = sq.OrderBy(orderBys...)
	}
	parts := strings.Split(q, " ")
	for _, part := range parts {
		sq = sq.Where(Like{"full_text": "% " + part + "%"})
	}
	err := r.queryAll(sq, results)
	return err
}

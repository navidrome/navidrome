package persistence

import (
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/kennygrant/sanitize"
)

func (r sqlRepository) getFullText(text ...string) string {
	sanitizedText := strings.Builder{}
	for _, txt := range text {
		sanitizedText.WriteString(strings.TrimSpace(sanitize.Accents(strings.ToLower(txt))) + " ")
	}
	return strings.TrimSpace(sanitizedText.String())
}

func (r sqlRepository) doSearch(q string, offset, size int, results interface{}, orderBys ...string) error {
	q = strings.TrimSpace(sanitize.Accents(strings.ToLower(strings.TrimSuffix(q, "*"))))
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
		sq = sq.Where(Or{
			Like{"full_text": part + "%"},
			Like{"full_text": "%" + part + "%"},
		})
	}
	err := r.queryAll(sq, results)
	return err
}

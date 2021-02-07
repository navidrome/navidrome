package persistence

import (
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/utils"
)

func getFullText(text ...string) string {
	fullText := utils.SanitizeStrings(text...)
	return " " + fullText
}

func (r sqlRepository) doSearch(q string, offset, size int, results interface{}, orderBys ...string) error {
	q = strings.TrimSpace(q)
	q = strings.TrimSuffix(q, "*")
	if len(q) < 2 {
		return nil
	}

	sq := r.newSelectWithAnnotation(r.tableName + ".id").Columns("*")
	sq = sq.Limit(uint64(size)).Offset(uint64(offset))
	if len(orderBys) > 0 {
		sq = sq.OrderBy(orderBys...)
	}
	sq = sq.Where(fullTextExpr(q))
	err := r.queryAll(sq, results)
	return err
}

func fullTextExpr(value string) Sqlizer {
	var sep string
	if !conf.Server.SearchFullString {
		sep = " "
	}
	q := utils.SanitizeStrings(value)
	parts := strings.Split(q, " ")
	filters := And{}
	for _, part := range parts {
		filters = append(filters, Like{"full_text": "%" + sep + part + "%"})
	}
	return filters
}

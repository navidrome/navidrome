package persistence

import (
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/str"
)

func getFullText(text ...string) string {
	fullText := str.SanitizeStrings(text...)
	return " " + fullText
}

func (r sqlRepository) doSearch(q string, offset, size int, results interface{}, orderBys ...string) error {
	q = strings.TrimSpace(q)
	q = strings.TrimSuffix(q, "*")
	if len(q) < 2 {
		return nil
	}

	sq := r.newSelectWithAnnotation(r.tableName + ".id").Columns(r.tableName + ".*")
	filter := fullTextExpr(q)
	if filter != nil {
		sq = sq.Where(filter)
		if len(orderBys) > 0 {
			sq = sq.OrderBy(orderBys...)
		}
	} else {
		// If the filter is empty, we sort by id.
		// This is to speed up the results of `search3?query=""`, for OpenSubsonic
		sq = sq.OrderBy("id")
	}
	sq = sq.Limit(uint64(size)).Offset(uint64(offset))
	err := r.queryAll(sq, results, model.QueryOptions{Offset: offset})
	return err
}

func fullTextExpr(value string) Sqlizer {
	q := str.SanitizeStrings(value)
	if q == "" {
		return nil
	}
	var sep string
	if !conf.Server.SearchFullString {
		sep = " "
	}
	parts := strings.Split(q, " ")
	filters := And{}
	for _, part := range parts {
		filters = append(filters, Like{"full_text": "%" + sep + part + "%"})
	}
	return filters
}

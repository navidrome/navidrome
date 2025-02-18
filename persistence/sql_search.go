package persistence

import (
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/str"
)

func formatFullText(text ...string) string {
	fullText := str.SanitizeStrings(text...)
	return " " + fullText
}

func (r sqlRepository) doSearch(sq SelectBuilder, q string, offset, size int, includeMissing bool, results any, orderBys ...string) error {
	q = strings.TrimSpace(q)
	q = strings.TrimSuffix(q, "*")
	if len(q) < 2 {
		return nil
	}

	//sq := r.newSelect().Columns(r.tableName + ".*")
	//sq = r.withAnnotation(sq, r.tableName+".id")
	//sq = r.withBookmark(sq, r.tableName+".id")
	filter := fullTextExpr(r.tableName, q)
	if filter != nil {
		sq = sq.Where(filter)
		sq = sq.OrderBy(orderBys...)
	} else {
		// If the filter is empty, we sort by rowid.
		// This is to speed up the results of `search3?query=""`, for OpenSubsonic
		sq = sq.OrderBy(r.tableName + ".rowid")
	}
	if !includeMissing {
		sq = sq.Where(Eq{r.tableName + ".missing": false})
	}
	sq = sq.Limit(uint64(size)).Offset(uint64(offset))
	return r.queryAll(sq, results, model.QueryOptions{Offset: offset})
}

func fullTextExpr(tableName string, s string) Sqlizer {
	q := str.SanitizeStrings(s)
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
		filters = append(filters, Like{tableName + ".full_text": "%" + sep + part + "%"})
	}
	return filters
}

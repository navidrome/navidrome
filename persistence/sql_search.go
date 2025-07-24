package persistence

import (
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
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

	filter := fullTextExpr(r.tableName, q)
	if filter != nil {
		sq = sq.Where(filter)
		sq = sq.OrderBy(orderBys...)
	} else {
		// If the filter is empty, we sort by id.
		// This is to speed up the results of `search3?query=""`, for OpenSubsonic
		sq = sq.OrderBy(r.tableName + ".id")
	}
	if !includeMissing {
		sq = sq.Where(Eq{r.tableName + ".missing": false})
	}
	sq = sq.Limit(uint64(size)).Offset(uint64(offset))
	return r.queryAll(sq, results, model.QueryOptions{Offset: offset})
}

func (r sqlRepository) searchByMBID(sq SelectBuilder, mbid string, mbidFields []string, includeMissing bool, results any) error {
	sq = sq.Where(mbidExpr(r.tableName, mbid, mbidFields...))

	if !includeMissing {
		sq = sq.Where(Eq{r.tableName + ".missing": false})
	}

	return r.queryAll(sq, results)
}

func mbidExpr(tableName, mbid string, mbidFields ...string) Sqlizer {
	if uuid.Validate(mbid) != nil || len(mbidFields) == 0 {
		return nil
	}
	mbid = strings.ToLower(mbid)
	var cond []Sqlizer
	for _, mbidField := range mbidFields {
		cond = append(cond, Eq{tableName + "." + mbidField: mbid})
	}
	return Or(cond)
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

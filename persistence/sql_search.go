package persistence

import (
	"fmt"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/str"
)

func formatFullText(text ...string) string {
	fullText := str.SanitizeStrings(text...)
	return " " + fullText
}

// searchExprFunc is the function signature for search expression builders.
type searchExprFunc func(tableName string, query string) Sqlizer

// getSearchExpr returns the active search expression function based on config.
func getSearchExpr() searchExprFunc {
	if conf.Server.Search.Backend == "legacy" || conf.Server.Search.FullString {
		return legacySearchExpr
	}
	return func(tableName, query string) Sqlizer {
		if containsCJK(query) {
			return likeSearchExpr(tableName, query)
		}
		return ftsSearchExpr(tableName, query)
	}
}

// doSearch performs a full-text search with the specified parameters.
// The naturalOrder is used when no filter is applied (e.g. OpenSubsonic `search3?query=""`).
// Normally it should be `tableName + ".rowid"`, but some repositories (e.g. artist) may differ.
func (r sqlRepository) doSearch(sq SelectBuilder, q string, offset, size int, results any, naturalOrder string, orderBys ...string) error {
	q = strings.TrimSpace(q)
	q = strings.TrimSuffix(q, "*")
	if len(q) < 2 {
		return nil
	}

	sq = sq.Where(Eq{r.tableName + ".missing": false})

	filter := getSearchExpr()(r.tableName, q)
	if filter == nil {
		// No search tokens; sort by natural order.
		sq = sq.OrderBy(naturalOrder)
		sq = sq.Limit(uint64(size)).Offset(uint64(offset))
		return r.queryAll(sq, results, model.QueryOptions{Offset: offset})
	}

	// Two-phase query for FTS5 to avoid expensive JOINs on high-cardinality results.
	if fts, ok := filter.(*ftsFilter); ok {
		return r.doFTSSearch(sq, fts, offset, size, results, orderBys...)
	}

	// LIKE/legacy: single-query approach.
	sq = sq.Where(filter)
	sq = sq.OrderBy(orderBys...)
	sq = sq.Limit(uint64(size)).Offset(uint64(offset))
	return r.queryAll(sq, results, model.QueryOptions{Offset: offset})
}

// doFTSSearch implements a two-phase FTS5 search to avoid expensive LEFT JOINs on
// high-cardinality FTS results.
//
// Phase 1: lightweight query (main table + FTS only) to get sorted, paginated rowids.
// Phase 2: full SELECT with all JOINs, filtered by the small set of Phase 1 rowids.
//
// Complex ORDER BY expressions (function calls, aggregations) are dropped from Phase 1;
// only FTS rank + simple columns are used.
func (r sqlRepository) doFTSSearch(sq SelectBuilder, fts *ftsFilter, offset, size int, results any, orderBys ...string) error {
	qualifiedOrderBys := []string{fts.rankExpr}
	for _, ob := range orderBys {
		if qualified := qualifyOrderBy(r.tableName, ob); qualified != "" {
			qualifiedOrderBys = append(qualifiedOrderBys, qualified)
		}
	}

	// Phase 1: only main table + FTS index, no annotation/bookmark/library JOINs.
	rowidQuery := Select(r.tableName+".rowid").
		From(r.tableName).
		Join(fts.ftsTable+" ON "+fts.ftsTable+".rowid = "+r.tableName+".rowid AND "+fts.ftsTable+" MATCH ?", fts.matchExpr).
		Where(Eq{r.tableName + ".missing": false}).
		OrderBy(qualifiedOrderBys...).
		Limit(uint64(size)).Offset(uint64(offset))

	rowidSQL, rowidArgs, err := rowidQuery.ToSql()
	if err != nil {
		return fmt.Errorf("building FTS rowid query: %w", err)
	}

	// Phase 2: hydrate with full JOINs, preserving Phase 1's ordering via row_number.
	rankedSubquery := fmt.Sprintf(
		"(SELECT rowid as _rid, row_number() OVER () AS _rn FROM (%s)) AS _ranked",
		rowidSQL,
	)
	sq = sq.Join(rankedSubquery+" ON "+r.tableName+".rowid = _ranked._rid", rowidArgs...)
	sq = sq.OrderBy("_ranked._rn")
	return r.queryAll(sq, results)
}

// qualifyOrderBy prepends tableName to a simple column name. Returns empty string for
// complex expressions (function calls, aggregations) that can't be used in Phase 1.
func qualifyOrderBy(tableName, orderBy string) string {
	orderBy = strings.TrimSpace(orderBy)
	if orderBy == "" || strings.ContainsAny(orderBy, "(,") {
		return ""
	}
	parts := strings.Fields(orderBy)
	if !strings.Contains(parts[0], ".") {
		parts[0] = tableName + "." + parts[0]
	}
	return strings.Join(parts, " ")
}

func (r sqlRepository) searchByMBID(sq SelectBuilder, mbid string, mbidFields []string, results any) error {
	sq = sq.Where(mbidExpr(r.tableName, mbid, mbidFields...))
	sq = sq.Where(Eq{r.tableName + ".missing": false})

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

// legacySearchExpr generates LIKE-based search filters against the full_text column.
// This is the original search implementation, used when Search.Backend="legacy".
func legacySearchExpr(tableName string, s string) Sqlizer {
	q := str.SanitizeStrings(s)
	if q == "" {
		log.Trace("Search using legacy backend, query is empty", "table", tableName)
		return nil
	}
	var sep string
	if !conf.Server.Search.FullString {
		sep = " "
	}
	parts := strings.Split(q, " ")
	filters := And{}
	for _, part := range parts {
		filters = append(filters, Like{tableName + ".full_text": "%" + sep + part + "%"})
	}
	log.Trace("Search using legacy backend", "query", filters, "table", tableName)
	return filters
}

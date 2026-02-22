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
// It falls back to legacySearchExpr when Search.FullString is enabled, because
// FTS5 is token-based and cannot match substrings within words.
// CJK queries are routed to likeSearchExpr, since FTS5's unicode61 tokenizer
// cannot segment CJK text.
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
// The naturalOrder is used to sort results when no full-text filter is applied. It is useful for cases like
// OpenSubsonic, where an empty search query should return all results in a natural order. Normally the parameter
// should be `tableName + ".rowid"`, but some repositories (ex: artist) may use a different natural order.
func (r sqlRepository) doSearch(sq SelectBuilder, q string, offset, size int, results any, naturalOrder string, orderBys ...string) error {
	q = strings.TrimSpace(q)
	q = strings.TrimSuffix(q, "*")
	if len(q) < 2 {
		return nil
	}

	searchExpr := getSearchExpr()
	filter := searchExpr(r.tableName, q)
	if filter == nil {
		// This is to speed up the results of `search3?query=""`, for OpenSubsonic
		// If the filter is empty, we sort by the specified natural order.
		sq = sq.OrderBy(naturalOrder)
		sq = sq.Where(Eq{r.tableName + ".missing": false})
		sq = sq.Limit(uint64(size)).Offset(uint64(offset))
		return r.queryAll(sq, results, model.QueryOptions{Offset: offset})
	}

	// For FTS5 filters, use a two-phase query to avoid expensive JOINs on high-cardinality results.
	if fts, ok := filter.(*ftsFilter); ok {
		return r.doFTSSearch(sq, fts, offset, size, results, orderBys...)
	}

	// For non-FTS filters (LIKE, legacy), use the original single-query approach.
	sq = sq.Where(filter)
	sq = sq.OrderBy(orderBys...)
	sq = sq.Where(Eq{r.tableName + ".missing": false})
	sq = sq.Limit(uint64(size)).Offset(uint64(offset))
	return r.queryAll(sq, results, model.QueryOptions{Offset: offset})
}

// doFTSSearch implements a two-phase FTS5 search to avoid the performance penalty of
// expensive LEFT JOINs (annotation, bookmark, library) on high-cardinality FTS results.
//
// Phase 1 builds a lightweight query that only touches the main table + FTS index to get
// the sorted, paginated rowids.
//
// Phase 2 uses the full SelectBuilder (with all JOINs) but
// filters by the small set of rowids from Phase 1, making the JOINs nearly free.
//
// If the ORDER BY references columns from other tables (e.g. aggregated stats), the two-phase
// approach is skipped and the original single-query approach is used instead.
func (r sqlRepository) doFTSSearch(sq SelectBuilder, fts *ftsFilter, offset, size int, results any, orderBys ...string) error {
	// Qualify ORDER BY columns for the lightweight query to avoid ambiguity between
	// the main table and the FTS table. Only simple column names (no expressions with
	// parens or commas) can be safely qualified; complex expressions indicate
	// the sort depends on JOINed tables, so we fall back to the single-query approach.
	qualifiedOrderBys := make([]string, 0, len(orderBys))
	for _, ob := range orderBys {
		qualified := qualifyOrderBy(r.tableName, ob)
		if qualified == "" {
			// Complex expression that can't be used in the lightweight query — fall back.
			sq = sq.Where(fts)
			sq = sq.OrderBy(orderBys...)
			sq = sq.Where(Eq{r.tableName + ".missing": false})
			sq = sq.Limit(uint64(size)).Offset(uint64(offset))
			return r.queryAll(sq, results, model.QueryOptions{Offset: offset})
		}
		qualifiedOrderBys = append(qualifiedOrderBys, qualified)
	}

	// Phase 1: Lightweight rowid query — only main table + FTS, no annotation/bookmark/library JOINs.
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

	// Phase 2: Hydrate the rowids with the full SelectBuilder (all JOINs included).
	sq = sq.Where(r.tableName+".rowid IN ("+rowidSQL+")", rowidArgs...)
	sq = sq.OrderBy(orderBys...)
	return r.queryAll(sq, results)
}

// qualifyOrderBy prepends tableName to a simple ORDER BY column name if it's not already
// qualified. Returns empty string for complex expressions (containing parens, commas, or
// function calls) that can't be safely used in a lightweight query without extra JOINs.
func qualifyOrderBy(tableName, orderBy string) string {
	orderBy = strings.TrimSpace(orderBy)
	if orderBy == "" {
		return ""
	}

	// If the expression contains parens (function calls like sum(...)) or commas,
	// it's too complex for the lightweight rowid query.
	if strings.ContainsAny(orderBy, "(,") {
		return ""
	}

	// Split into column and optional direction (e.g., "title" or "name desc")
	parts := strings.Fields(orderBy)
	col := parts[0]

	// Already qualified (contains a dot)
	if strings.Contains(col, ".") {
		return orderBy
	}

	// Qualify with table name
	parts[0] = tableName + "." + col
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

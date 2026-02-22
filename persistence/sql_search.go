package persistence

import (
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

// searchFilter is the internal result from search expression builders.
// It is never exposed to callers — they use getSearchFilter or applySearchFilter instead.
type searchFilter struct {
	where     Sqlizer // WHERE clause (LIKE/legacy/FTS5 rowid IN)
	rankOrder string  // ORDER BY expression for relevance (correlated bm25 subquery)
	rankArgs  []any   // Args for the rank ORDER BY expression
}

// buildSearchFilter returns the search filter for the given table and query,
// selecting the appropriate backend based on config (FTS5, legacy LIKE, CJK LIKE).
func buildSearchFilter(tableName, query string) *searchFilter {
	if conf.Server.Search.Backend == "legacy" || conf.Server.Search.FullString {
		return legacySearchExpr(tableName, query)
	}
	if containsCJK(query) {
		return likeSearchExpr(tableName, query)
	}
	return ftsSearchExpr(tableName, query)
}

// getSearchFilter returns a Sqlizer for WHERE-only filtering.
// Used by fullTextFilter where only filtering is needed, not ranking.
func getSearchFilter(tableName, query string) Sqlizer {
	filter := buildSearchFilter(tableName, query)
	if filter == nil {
		return nil
	}
	return filter.where
}

// applySearchFilter applies search filtering and ordering to a query builder.
// When a filter matches, it adds the WHERE clause, optional BM25 ranking, and orderBys as tiebreakers.
// When no filter matches (empty query), it falls back to naturalOrder.
func applySearchFilter(sq SelectBuilder, tableName, query, naturalOrder string, orderBys ...string) SelectBuilder {
	filter := buildSearchFilter(tableName, query)
	if filter == nil {
		return sq.OrderBy(naturalOrder)
	}
	sq = sq.Where(filter.where)
	if filter.rankOrder != "" {
		rankArgs := make([]interface{}, len(filter.rankArgs))
		copy(rankArgs, filter.rankArgs)
		sq = sq.OrderByClause(filter.rankOrder, rankArgs...)
	}
	return sq.OrderBy(orderBys...)
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

	sq = applySearchFilter(sq, r.tableName, q, naturalOrder, orderBys...)
	sq = sq.Where(Eq{r.tableName + ".missing": false})
	sq = sq.Limit(uint64(size)).Offset(uint64(offset))
	return r.queryAll(sq, results, model.QueryOptions{Offset: offset})
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
func legacySearchExpr(tableName string, s string) *searchFilter {
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
	return &searchFilter{where: filters}
}

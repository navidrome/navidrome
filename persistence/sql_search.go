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

// searchFilter carries the result of a search expression builder.
// For WHERE-based filters (legacy LIKE, CJK LIKE), only Where is set.
// For FTS5 ranked search, Where contains a rowid IN subquery for filtering,
// and RankOrder contains a correlated subquery for BM25 relevance ordering.
type searchFilter struct {
	Where     Sqlizer // WHERE clause (LIKE/legacy/FTS5 rowid IN)
	RankOrder string  // ORDER BY expression for relevance (correlated bm25 subquery)
	RankArgs  []any   // Args for the rank ORDER BY expression
}

// AsSqlizer returns a Sqlizer suitable for use in a WHERE clause.
// This is used in contexts like fullTextFilter where only filtering is needed, not ranking.
func (sf *searchFilter) AsSqlizer() Sqlizer {
	if sf == nil {
		return nil
	}
	return sf.Where
}

// searchExprFunc is the function signature for search expression builders.
type searchExprFunc func(tableName string, query string) *searchFilter

// getSearchExpr returns the active search expression function based on config.
// It falls back to legacySearchExpr when Search.FullString is enabled, because
// FTS5 is token-based and cannot match substrings within words.
// CJK queries are routed to likeSearchExpr, since FTS5's unicode61 tokenizer
// cannot segment CJK text.
func getSearchExpr() searchExprFunc {
	if conf.Server.Search.Backend == "legacy" || conf.Server.Search.FullString {
		return legacySearchExpr
	}
	return func(tableName, query string) *searchFilter {
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
	if filter != nil {
		sq = sq.Where(filter.Where)
		if filter.RankOrder != "" {
			// FTS5 ranked search: use correlated subquery for BM25 relevance ordering.
			// OrderByClause supports parameterized args (unlike OrderBy).
			rankArgs := make([]interface{}, len(filter.RankArgs))
			copy(rankArgs, filter.RankArgs)
			sq = sq.OrderByClause(filter.RankOrder, rankArgs...)
			sq = sq.OrderBy(orderBys...)
		} else {
			// WHERE-based search (legacy LIKE, CJK LIKE): no ranking
			sq = sq.OrderBy(orderBys...)
		}
	} else {
		// This is to speed up the results of `search3?query=""`, for OpenSubsonic
		// If the filter is empty, we sort by the specified natural order.
		sq = sq.OrderBy(naturalOrder)
	}
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
	return &searchFilter{Where: filters}
}

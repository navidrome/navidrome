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

// searchConfig holds per-repository constants for doSearch.
type searchConfig struct {
	NaturalOrder string   // ORDER BY for empty-query results (e.g. "album.rowid")
	OrderBy      []string // ORDER BY for text search results (e.g. ["name"])
	MBIDFields   []string // columns to match when query is a UUID
	// LibraryFilter overrides the default applyLibraryFilter for FTS Phase 1.
	// Needed when library access requires a junction table (e.g. artist → library_artist).
	LibraryFilter func(sq SelectBuilder) SelectBuilder
}

// searchStrategy defines how to execute a text search against a repository table.
// options carries filters and pagination that must reach all query phases,
// including FTS Phase 1 which builds its own query outside sq.
type searchStrategy interface {
	Sqlizer
	execute(r sqlRepository, sq SelectBuilder, dest any, cfg searchConfig, options model.QueryOptions) error
}

// getSearchStrategy returns the appropriate search strategy based on config and query content.
// Returns nil when the query produces no searchable tokens.
func getSearchStrategy(tableName, query string) searchStrategy {
	if conf.Server.Search.Backend == "legacy" || conf.Server.Search.FullString {
		return newLegacySearch(tableName, query)
	}
	if containsCJK(query) {
		return newLikeSearch(tableName, query)
	}
	return newFTSSearch(tableName, query)
}

// doSearch dispatches a search query: empty → natural order, UUID → MBID match,
// otherwise delegates to getSearchStrategy. sq must already have LIMIT/OFFSET set
// via newSelect(options...). options is forwarded so FTS Phase 1 can apply the same
// filters and pagination independently.
func (r sqlRepository) doSearch(sq SelectBuilder, q string, results any, cfg searchConfig, options model.QueryOptions) error {
	q = strings.TrimSpace(q)
	q = strings.TrimSuffix(q, "*")

	sq = sq.Where(Eq{r.tableName + ".missing": false})

	// Empty query (OpenSubsonic `search3?query=""`) — return all in natural order.
	if q == "" || q == `""` {
		sq = sq.OrderBy(cfg.NaturalOrder)
		return r.queryAll(sq, results, options)
	}

	// MBID search: if query is a valid UUID, search by MBID fields instead
	if uuid.Validate(q) == nil && len(cfg.MBIDFields) > 0 {
		sq = sq.Where(mbidExpr(r.tableName, q, cfg.MBIDFields...))
		return r.queryAll(sq, results)
	}

	// Min-length guard: single-character queries are too broad for search3.
	// This check lives here (not in the strategies) so that fullTextFilter
	// (REST filter path) can still use single-character queries.
	if len(q) < 2 {
		return nil
	}

	strategy := getSearchStrategy(r.tableName, q)
	if strategy == nil {
		return nil
	}

	return strategy.execute(r, sq, results, cfg, options)
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

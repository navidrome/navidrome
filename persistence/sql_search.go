package persistence

import (
	"fmt"
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
	// LibraryFilter overrides the default applyLibraryFilter for the rowid Phase 1 of
	// two-phase searches (FTS and empty-query). Needed when library access goes through a
	// junction table (e.g. artist → library_artist), whose JOIN can fan out rowids for
	// entities in multiple libraries — Phase 1 dedups whenever this is set.
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
		rowidCore := Select(r.tableName + ".rowid").From(r.tableName).OrderBy(cfg.NaturalOrder)
		return r.executeTwoPhase(sq, results, rowidCore, cfg, options)
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

// executeTwoPhase runs a search in two phases:
//   - Phase 1: rowidCore (strategy-specific FROM/JOINs and ORDER BY) plus the shared search
//     contract applied here — non-missing rows only, library access, options.Filters, and
//     pagination. Keeping Phase 1 free of the full SELECT's JOINs lets SQLite paginate via a
//     covering index; with those JOINs, large offsets degrade to O(offset) join probes —
//     multi-second responses on 100k+ libraries.
//   - Phase 2: full SELECT with all JOINs, scoped to Phase 1's rowid page.
func (r sqlRepository) executeTwoPhase(sq SelectBuilder, results any, rowidCore SelectBuilder, cfg searchConfig, options model.QueryOptions) error {
	rowidQuery := rowidCore.
		Where(Eq{r.tableName + ".missing": false})
	if options.Max > 0 {
		rowidQuery = rowidQuery.Limit(uint64(options.Max))
	}
	if options.Offset > 0 {
		rowidQuery = rowidQuery.Offset(uint64(options.Offset))
	}
	if cfg.LibraryFilter != nil {
		// Junction-table library filters can repeat rowids for entities in multiple
		// libraries, which would corrupt offset-based pagination — dedup before paginating.
		// (DISTINCT, not GROUP BY: bm25() can't be evaluated in a grouped query.)
		rowidQuery = cfg.LibraryFilter(rowidQuery).Distinct()
	} else {
		rowidQuery = r.applyLibraryFilter(rowidQuery)
	}
	if options.Filters != nil {
		rowidQuery = rowidQuery.Where(options.Filters)
	}
	return r.hydrateRowidPage(sq, rowidQuery, results)
}

// hydrateRowidPage joins sq to the ordered rowid set produced by rowidQuery, preserving its
// ordering. rowidQuery must handle pagination itself; sq's LIMIT/OFFSET are stripped.
func (r sqlRepository) hydrateRowidPage(sq SelectBuilder, rowidQuery SelectBuilder, results any) error {
	rowidSQL, rowidArgs, err := rowidQuery.ToSql()
	if err != nil {
		return fmt.Errorf("building rowid query: %w", err)
	}
	sq = sq.RemoveLimit().RemoveOffset()
	rankedSubquery := fmt.Sprintf(
		"(SELECT rowid as _rid, row_number() OVER () AS _rn FROM (%s)) AS _ranked",
		rowidSQL,
	)
	sq = sq.Join(rankedSubquery+" ON "+r.tableName+".rowid = _ranked._rid", rowidArgs...)
	sq = sq.OrderBy("_ranked._rn")
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

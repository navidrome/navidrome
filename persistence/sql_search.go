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
		return r.executeNaturalSearch(sq, results, cfg, options)
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

// executeNaturalSearch returns all non-missing rows in natural order, for the empty-query
// case (`search3?query=""`), which clients like Symfonium use to paginate over the whole
// library when syncing. It runs in two phases, like ftsSearch.execute:
//   - Phase 1: rowid-only query on the main table (no JOINs), so SQLite can paginate via a
//     covering index. With the JOINs of the full SELECT, large offsets degrade to O(offset)
//     join probes — multi-second responses on 100k+ libraries (the whole table is walked).
//   - Phase 2: full SELECT with all JOINs, scoped to Phase 1's rowid page.
func (r sqlRepository) executeNaturalSearch(sq SelectBuilder, results any, cfg searchConfig, options model.QueryOptions) error {
	rowidQuery := Select(r.tableName + ".rowid").
		From(r.tableName).
		Where(Eq{r.tableName + ".missing": false}).
		OrderBy(cfg.NaturalOrder)
	if options.Max > 0 {
		rowidQuery = rowidQuery.Limit(uint64(options.Max))
	}
	if options.Offset > 0 {
		rowidQuery = rowidQuery.Offset(uint64(options.Offset))
	}
	if cfg.LibraryFilter != nil {
		rowidQuery = cfg.LibraryFilter(rowidQuery)
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

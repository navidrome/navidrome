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

// searchStrategy defines how to execute a text search against a repository table.
// ToSql() (from embedded Sqlizer) is used by the REST filter path as a WHERE clause.
// execute() is used by the Search endpoints for full search with pagination and ordering.
type searchStrategy interface {
	Sqlizer
	execute(r sqlRepository, sq SelectBuilder, offset, size int, dest any, orderBys ...string) error
}

// getSearchStrategy returns the appropriate search strategy based on config and query content.
// Returns nil when the query is too short for the selected strategy or produces no searchable tokens.
func getSearchStrategy(tableName, query string) searchStrategy {
	if conf.Server.Search.Backend == "legacy" || conf.Server.Search.FullString {
		return newLegacySearch(tableName, query)
	}
	if containsCJK(query) {
		return newLikeSearch(tableName, query)
	}
	return newFTSSearch(tableName, query)
}

// doSearch performs a full-text search with the specified parameters.
// Empty queries return all results in natural order (OpenSubsonic `search3?query=""`).
// The naturalOrder column should normally be `tableName + ".rowid"`, but some repositories
// (e.g. artist) may differ. Minimum query length is enforced by each strategy individually.
func (r sqlRepository) doSearch(sq SelectBuilder, q string, offset, size int, results any, naturalOrder string, orderBys ...string) error {
	q = strings.TrimSpace(q)
	q = strings.TrimSuffix(q, "*")

	sq = sq.Where(Eq{r.tableName + ".missing": false})

	// Empty or quoted-empty query (OpenSubsonic `search3?query=""`) — return all results in natural order.
	if q == "" || q == `""` {
		sq = sq.OrderBy(naturalOrder)
		sq = sq.Limit(uint64(size)).Offset(uint64(offset))
		return r.queryAll(sq, results, model.QueryOptions{Offset: offset})
	}

	strategy := getSearchStrategy(r.tableName, q)
	if strategy == nil {
		return nil
	}

	return strategy.execute(r, sq, offset, size, results, orderBys...)
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

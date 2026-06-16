package persistence

import (
	"database/sql/driver"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/fatih/structs"
)

type PostMapper interface {
	PostMapArgs(map[string]any) error
}

func toSQLArgs(rec any) (map[string]any, error) {
	m := structs.Map(rec)
	for k, v := range m {
		switch t := v.(type) {
		case *time.Time:
			if t != nil {
				m[k] = *t
			}
		case driver.Valuer:
			var err error
			m[k], err = t.Value()
			if err != nil {
				return nil, err
			}
		}
	}
	if r, ok := rec.(PostMapper); ok {
		err := r.PostMapArgs(m)
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

var matchUnderscore = regexp.MustCompile("_([A-Za-z])")

func toCamelCase(str string) string {
	return matchUnderscore.ReplaceAllStringFunc(str, func(s string) string {
		return strings.ToUpper(strings.Replace(s, "_", "", -1))
	})
}

func Exists(subTable string, cond squirrel.Sqlizer) existsCond {
	return existsCond{subTable: subTable, cond: cond, not: false}
}

// ArtistLibraryFilter restricts artists to the given libraries via a correlated EXISTS over the
// library_artist junction. It is join-free on purpose: the search Phase 1 paginates rowids ordered
// by artist.id, and a CROSS JOIN would fan out rowids (forcing a DISTINCT temp b-tree that makes
// deep offsets O(offset)). EXISTS keeps artist as the ordered scan driver, so LIMIT/OFFSET
// short-circuits — O(page).
//
// The inner LIMIT 1 is load-bearing: without it SQLite may flatten the correlated EXISTS into a
// join that fans out — an artist in N of the libraries then yields N rowids, duplicating/skipping
// paginated rows. A subquery with a LIMIT cannot be flattened, so the LIMIT 1 forces the semi-join
// (boolean) evaluation while staying an index seek on the (library_id, artist_id) UNIQUE autoindex.
func ArtistLibraryFilter(libraryIDs []int) squirrel.Sqlizer {
	if len(libraryIDs) == 0 {
		return squirrel.Eq{"1": 2} // match nothing, without a degenerate `IN ()` subquery
	}
	sub, args, _ := squirrel.Select("1").From("library_artist").
		Where(squirrel.And{
			squirrel.Expr("library_artist.artist_id = artist.id"),
			squirrel.Eq{"library_artist.library_id": libraryIDs},
		}).Limit(1).ToSql()
	return squirrel.Expr("EXISTS ("+sub+")", args...)
}

func NotExists(subTable string, cond squirrel.Sqlizer) existsCond {
	return existsCond{subTable: subTable, cond: cond, not: true}
}

type existsCond struct {
	subTable string
	cond     squirrel.Sqlizer
	not      bool
}

func (e existsCond) ToSql() (string, []any, error) {
	sql, args, err := e.cond.ToSql()
	sql = fmt.Sprintf("exists (select 1 from %s where %s)", e.subTable, sql)
	if e.not {
		sql = "not " + sql
	}
	return sql, args, err
}

var sortOrderRegex = regexp.MustCompile(`order_([a-z_]+)`)

// Convert the order_* columns to an expression using sort_* columns. Example:
// sort_album_name -> (coalesce(nullif(sort_album_name,”),order_album_name) collate nocase)
// It finds order column names anywhere in the substring
func mapSortOrder(tableName, order string) string {
	order = strings.ToLower(order)
	repl := fmt.Sprintf("(coalesce(nullif(%[1]s.sort_$1,''),%[1]s.order_$1) collate nocase)", tableName)
	return sortOrderRegex.ReplaceAllString(order, repl)
}

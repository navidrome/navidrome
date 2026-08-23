package persistence

import (
	"database/sql/driver"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/fatih/structs"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/db"
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

func sortCollation() string {
	if conf.Server.EnableNaturalSorting {
		return db.NaturalCollation
	}
	return "nocase"
}

// collatedSort wraps a plain text column so it sorts with the configured
// collation. The parens are required: buildSortOrder splits on spaces.
func collatedSort(col string) string {
	return fmt.Sprintf("(%s collate %s)", col, sortCollation())
}

// mapNaturalOrder qualifies the order_* columns and applies the natural collation.
// Used when sort_* columns are not preferred, so mapSortOrder does not run.
func mapNaturalOrder(tableName, order string) string {
	order = strings.ToLower(order)
	repl := fmt.Sprintf("(%[1]s.order_$1 collate %[2]s)", tableName, db.NaturalCollation)
	return sortOrderRegex.ReplaceAllString(order, repl)
}

// Convert the order_* columns to an expression using sort_* columns. Example:
// sort_album_name -> (coalesce(nullif(sort_album_name,”),order_album_name) collate nocase)
// It finds order column names anywhere in the substring
func mapSortOrder(tableName, order string) string {
	order = strings.ToLower(order)
	repl := fmt.Sprintf("(coalesce(nullif(%[1]s.sort_$1,''),%[1]s.order_$1) collate %[2]s)", tableName, sortCollation())
	return sortOrderRegex.ReplaceAllString(order, repl)
}

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

func toSQLArgs(rec interface{}) (map[string]interface{}, error) {
	m := structs.Map(rec)
	for k, v := range m {
		switch t := v.(type) {
		case time.Time:
			m[k] = t.Format(time.RFC3339Nano)
		case *time.Time:
			if t != nil {
				m[k] = t.Format(time.RFC3339Nano)
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

func exists(subTable string, cond squirrel.Sqlizer) existsCond {
	return existsCond{subTable: subTable, cond: cond, not: false}
}

func notExists(subTable string, cond squirrel.Sqlizer) existsCond {
	return existsCond{subTable: subTable, cond: cond, not: true}
}

type existsCond struct {
	subTable string
	cond     squirrel.Sqlizer
	not      bool
}

func (e existsCond) ToSql() (string, []interface{}, error) {
	sql, args, err := e.cond.ToSql()
	sql = fmt.Sprintf("exists (select 1 from %s where %s)", e.subTable, sql)
	if e.not {
		sql = "not " + sql
	}
	return sql, args, err
}

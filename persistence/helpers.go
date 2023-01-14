package persistence

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/fatih/structs"
)

func toSqlArgs(rec interface{}) (map[string]interface{}, error) {
	m := structs.Map(rec)
	for k, v := range m {
		if t, ok := v.(time.Time); ok {
			m[k] = t.Format(time.RFC3339Nano)
		}
		if t, ok := v.(*time.Time); ok && t != nil {
			m[k] = t.Format(time.RFC3339Nano)
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

package persistence

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

func toSqlArgs(rec interface{}) (map[string]interface{}, error) {
	// Convert to JSON...
	b, err := json.Marshal(rec)
	if err != nil {
		return nil, err
	}

	// ... then convert to map
	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	r := make(map[string]interface{}, len(m))
	for f, v := range m {
		r[toSnakeCase(f)] = v
	}
	return r, err
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

type exist string

func (e exist) ToSql() (string, []interface{}, error) {
	sql := fmt.Sprintf("exists (select 1 %s)", e)
	return sql, nil, nil
}

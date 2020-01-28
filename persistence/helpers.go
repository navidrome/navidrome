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

func ToStruct(m map[string]interface{}, rec interface{}, fieldNames []string) error {
	var r = make(map[string]interface{}, len(m))
	for _, f := range fieldNames {
		v, ok := m[f]
		if !ok {
			return fmt.Errorf("invalid field '%s'", f)
		}
		r[toCamelCase(f)] = v
	}
	// Convert to JSON...
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}

	// ... then convert to struct
	err = json.Unmarshal(b, &rec)
	return err
}

var matchUnderscore = regexp.MustCompile("_([A-Za-z])")

func toCamelCase(str string) string {
	return matchUnderscore.ReplaceAllStringFunc(str, func(s string) string {
		return strings.ToUpper(strings.Replace(s, "_", "", -1))
	})
}

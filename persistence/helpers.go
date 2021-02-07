package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
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
		isAnnotationField := utils.StringInSlice(f, model.AnnotationFields)
		isBookmarkField := utils.StringInSlice(f, model.BookmarkFields)
		if !isAnnotationField && !isBookmarkField && v != nil {
			r[toSnakeCase(f)] = v
		}
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

func exists(subTable string, cond squirrel.Sqlizer) existsCond {
	return existsCond{subTable: subTable, cond: cond}
}

type existsCond struct {
	subTable string
	cond     squirrel.Sqlizer
}

func (e existsCond) ToSql() (string, []interface{}, error) {
	sql, args, err := e.cond.ToSql()
	sql = fmt.Sprintf("exists (select 1 from %s where %s)", e.subTable, sql)
	return sql, args, err
}

func getMbzId(ctx context.Context, mbzIDS, entityName, name string) string {
	ids := strings.Fields(mbzIDS)
	if len(ids) == 0 {
		return ""
	}
	idCounts := map[string]int{}
	for _, id := range ids {
		if c, ok := idCounts[id]; ok {
			idCounts[id] = c + 1
		} else {
			idCounts[id] = 1
		}
	}

	var topKey string
	var topCount int
	for k, v := range idCounts {
		if v > topCount {
			topKey = k
			topCount = v
		}
	}

	if len(idCounts) > 1 && name != consts.VariousArtists {
		log.Warn(ctx, "Multiple MBIDs found for "+entityName, "name", name, "mbids", idCounts)
	}
	return topKey
}

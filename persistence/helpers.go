package persistence

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/fatih/structs"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
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

func getMostFrequentMbzID(ctx context.Context, mbzIDs, entityName, name string) string {
	ids := strings.Fields(mbzIDs)
	if len(ids) == 0 {
		return ""
	}
	var topId string
	var topCount int
	idCounts := map[string]int{}

	if len(ids) == 1 {
		topId = ids[0]
	} else {
		for _, id := range ids {
			c := idCounts[id] + 1
			idCounts[id] = c
			if c > topCount {
				topId = id
				topCount = c
			}
		}
	}

	if len(idCounts) > 1 && name != consts.VariousArtists {
		log.Warn(ctx, "Multiple MBIDs found for "+entityName, "name", name, "mbids", idCounts, "selectedId", topId)
	}
	if topId == consts.VariousArtistsMbzId && name != consts.VariousArtists {
		log.Warn(ctx, "Artist with mbid of 'Various Artists'", "name", name, "mbid", topId)
	}

	return topId
}

func getGenres(genreIds string) model.Genres {
	ids := strings.Fields(genreIds)
	var genres model.Genres
	unique := map[string]struct{}{}
	for _, id := range ids {
		if _, ok := unique[id]; ok {
			continue
		}
		genres = append(genres, model.Genre{ID: id})
		unique[id] = struct{}{}
	}
	return genres
}

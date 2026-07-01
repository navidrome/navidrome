package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/fatih/structs"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

const annotationTable = "annotation"

// annotationColumns are the columns withAnnotation's LEFT JOIN contributes, derived from
// model.Annotations so the set tracks schema changes. average_rating is excluded: it lives on the
// base table, not the annotation join.
var annotationColumns = sync.OnceValue(func() map[string]struct{} {
	cols := map[string]struct{}{}
	for name := range structs.Map(model.Annotations{}) {
		if name == "average_rating" {
			continue
		}
		cols[name] = struct{}{}
	}
	return cols
})

// annotationColumnRE matches any annotation column as a whole word. The word boundaries keep the
// base-table column average_rating from matching the annotation column rating (Go's \b treats '_'
// as a word char). It is case-insensitive because SQLite column names are, so a raw filter using
// e.g. "RATING" must still be detected.
var annotationColumnRE = sync.OnceValue(func() *regexp.Regexp {
	cols := make([]string, 0, len(annotationColumns()))
	for col := range annotationColumns() {
		cols = append(cols, regexp.QuoteMeta(col))
	}
	sort.Strings(cols) // map iteration is random; sort for a stable pattern
	return regexp.MustCompile(`(?i)\b(?:` + strings.Join(cols, "|") + `)\b`)
})

// filtersNeedAnnotation reports whether the rendered query references an annotation column, i.e.
// whether the annotation LEFT JOIN must be kept. Scanning the rendered SQL catches every filter
// path. The placeholder column is needed because squirrel won't render a column-less SELECT; on a
// render error, keep the join to be safe.
func filtersNeedAnnotation(query SelectBuilder) bool {
	sql, _, err := query.Columns("1").ToSql()
	if err != nil {
		return true
	}
	return annotationColumnRE().MatchString(sql)
}

func (r sqlRepository) withAnnotation(query SelectBuilder, idField string) SelectBuilder {
	userID := loggedUser(r.ctx).ID
	if userID == invalidUserId {
		return query.Columns(fmt.Sprintf("%s.average_rating", r.tableName))
	}
	query = query.
		LeftJoin("annotation on ("+
			"annotation.item_id = "+idField+
			" AND annotation.user_id = '"+userID+"')").
		Columns(
			"coalesce(starred, 0) as starred",
			"coalesce(rating, 0) as rating",
			"starred_at",
			"play_date",
			"rated_at",
		)
	if conf.Server.AlbumPlayCountMode == consts.AlbumPlayCountModeNormalized && r.tableName == "album" {
		query = query.Columns(
			fmt.Sprintf("round(coalesce(round(cast(play_count as float) / coalesce(%[1]s.song_count, 1), 1), 0)) as play_count", r.tableName),
		)
	} else {
		query = query.Columns("coalesce(play_count, 0) as play_count")
	}

	query = query.Columns(fmt.Sprintf("%s.average_rating", r.tableName))

	return query
}

func annotationBoolFilter(field string) func(string, any) Sqlizer {
	return func(_ string, value any) Sqlizer {
		v, ok := value.(string)
		if !ok {
			return nil
		}
		if strings.ToLower(v) == "true" {
			return Expr(fmt.Sprintf("COALESCE(%s, 0) > 0", field))
		}
		return Expr(fmt.Sprintf("COALESCE(%s, 0) = 0", field))
	}
}

func (r sqlRepository) annId(itemID ...string) And {
	userID := loggedUser(r.ctx).ID
	return And{
		Eq{annotationTable + ".user_id": userID},
		Eq{annotationTable + ".item_type": r.tableName},
		Eq{annotationTable + ".item_id": itemID},
	}
}

func (r sqlRepository) annUpsert(values map[string]any, itemIDs ...string) error {
	upd := Update(annotationTable).Where(r.annId(itemIDs...))
	for f, v := range values {
		upd = upd.Set(f, v)
	}
	c, err := r.executeSQL(upd)
	if c == 0 || errors.Is(err, sql.ErrNoRows) {
		userID := loggedUser(r.ctx).ID
		for _, itemID := range itemIDs {
			values["user_id"] = userID
			values["item_type"] = r.tableName
			values["item_id"] = itemID
			ins := Insert(annotationTable).SetMap(values)
			_, err = r.executeSQL(ins)
			if err != nil {
				return err
			}
		}
	}
	return err
}

func (r sqlRepository) SetStar(starred bool, ids ...string) error {
	starredAt := time.Now()
	return r.annUpsert(map[string]any{"starred": starred, "starred_at": starredAt}, ids...)
}

func (r sqlRepository) SetRating(rating int, itemID string) error {
	ratedAt := time.Now()
	err := r.annUpsert(map[string]any{"rating": rating, "rated_at": ratedAt}, itemID)
	if err != nil {
		return err
	}
	return r.updateAvgRating(itemID)
}

func (r sqlRepository) updateAvgRating(itemID string) error {
	upd := Update(r.tableName).
		Where(Eq{"id": itemID}).
		Set("average_rating", Expr(
			"coalesce((select round(avg(rating), 2) from annotation where item_id = ? and item_type = ? and rating > 0), 0)",
			itemID, r.tableName,
		))
	_, err := r.executeSQL(upd)
	return err
}

func (r sqlRepository) IncPlayCount(itemID string, ts time.Time) error {
	upd := Update(annotationTable).Where(r.annId(itemID)).
		Set("play_count", Expr("play_count+1")).
		Set("play_date", Expr("max(ifnull(play_date,''),?)", ts))
	c, err := r.executeSQL(upd)

	if c == 0 || errors.Is(err, sql.ErrNoRows) {
		userID := loggedUser(r.ctx).ID
		values := map[string]any{}
		values["user_id"] = userID
		values["item_type"] = r.tableName
		values["item_id"] = itemID
		values["play_count"] = 1
		values["play_date"] = ts
		ins := Insert(annotationTable).SetMap(values)
		_, err = r.executeSQL(ins)
		if err != nil {
			return err
		}
	}
	return err
}

func (r sqlRepository) ReassignAnnotation(prevID string, newID string) error {
	if prevID == newID || prevID == "" || newID == "" {
		return nil
	}
	upd := Update(annotationTable).Where(And{
		Eq{annotationTable + ".item_type": r.tableName},
		Eq{annotationTable + ".item_id": prevID},
	}).Set("item_id", newID)
	_, err := r.executeSQL(upd)
	return err
}

func (r sqlRepository) cleanAnnotations() error {
	del := Delete(annotationTable).Where(Eq{"item_type": r.tableName}).Where("item_id not in (select id from " + r.tableName + ")")
	c, err := r.executeSQL(del)
	if err != nil {
		return fmt.Errorf("error cleaning up %s annotations: %w", r.tableName, err)
	}
	if c > 0 {
		log.Debug(r.ctx, "Clean-up annotations", "table", r.tableName, "totalDeleted", c)
	}
	return nil
}

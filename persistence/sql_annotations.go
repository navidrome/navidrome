package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
)

const annotationTable = "annotation"

func (r sqlRepository) withAnnotation(query SelectBuilder, idField string) SelectBuilder {
	userID := loggedUser(r.ctx).ID
	if userID == invalidUserId {
		return query
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
		)
	if conf.Server.AlbumPlayCountMode == consts.AlbumPlayCountModeNormalized && r.tableName == "album" {
		query = query.Columns(
			fmt.Sprintf("round(coalesce(round(cast(play_count as float) / coalesce(%[1]s.song_count, 1), 1), 0)) as play_count", r.tableName),
		)
	} else {
		query = query.Columns("coalesce(play_count, 0) as play_count")
	}

	return query
}

func (r sqlRepository) annId(itemID ...string) And {
	userID := loggedUser(r.ctx).ID
	return And{
		Eq{annotationTable + ".user_id": userID},
		Eq{annotationTable + ".item_type": r.tableName},
		Eq{annotationTable + ".item_id": itemID},
	}
}

func (r sqlRepository) annUpsert(values map[string]interface{}, itemIDs ...string) error {
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
	return r.annUpsert(map[string]interface{}{"starred": starred, "starred_at": starredAt}, ids...)
}

func (r sqlRepository) SetRating(rating int, itemID string) error {
	return r.annUpsert(map[string]interface{}{"rating": rating}, itemID)
}

func (r sqlRepository) IncPlayCount(itemID string, ts time.Time) error {
	upd := Update(annotationTable).Where(r.annId(itemID)).
		Set("play_count", Expr("play_count+1")).
		Set("play_date", Expr("max(ifnull(play_date,''),?)", ts))
	c, err := r.executeSQL(upd)

	if c == 0 || errors.Is(err, sql.ErrNoRows) {
		userID := loggedUser(r.ctx).ID
		values := map[string]interface{}{}
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
		return fmt.Errorf("error cleaning up annotations: %w", err)
	}
	if c > 0 {
		log.Debug(r.ctx, "Clean-up annotations", "table", r.tableName, "totalDeleted", c)
	}
	return nil
}

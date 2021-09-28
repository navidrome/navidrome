package persistence

import (
	"database/sql"
	"errors"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

const annotationTable = "annotation"

func (r sqlRepository) newSelectWithAnnotation(idField string, options ...model.QueryOptions) SelectBuilder {
	query := r.newSelect(options...).
		LeftJoin("annotation on ("+
			"annotation.item_id = "+idField+
			" AND annotation.item_type = '"+r.tableName+"'"+
			" AND annotation.user_id = '"+userId(r.ctx)+"')").
		Columns(
			"coalesce(starred, 0) as starred",
			"coalesce(rating, 0) as rating",
			"starred_at",
			"play_date",
		)
	if conf.Server.AlbumPlayCountMode == consts.AlbumPlayCountModeNormalized && r.tableName == "album" {
		query = query.Columns("round(coalesce(round(cast(play_count as float) / coalesce(song_count, 1), 1), 0)) as play_count")
	} else {
		query = query.Columns("coalesce(play_count, 0) as play_count")
	}

	return query
}

func (r sqlRepository) annId(itemID ...string) And {
	return And{
		Eq{annotationTable + ".user_id": userId(r.ctx)},
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
		for _, itemID := range itemIDs {
			values["user_id"] = userId(r.ctx)
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
		values := map[string]interface{}{}
		values["user_id"] = userId(r.ctx)
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

func (r sqlRepository) cleanAnnotations() error {
	del := Delete(annotationTable).Where(Eq{"item_type": r.tableName}).Where("item_id not in (select id from " + r.tableName + ")")
	c, err := r.executeSQL(del)
	if err != nil {
		return err
	}
	if c > 0 {
		log.Debug(r.ctx, "Clean-up annotations", "table", r.tableName, "totalDeleted", c)
	}
	return nil
}

func (r sqlRepository) CopyAnnotation(fromID string, toID string) error {
	if fromID == toID {
		return nil
	}

	/* Delete existing ones so we don't get conflicts. */
	del := Delete(annotationTable).Where(And{
		Eq{annotationTable + ".item_type": r.tableName},
		Eq{annotationTable + ".item_id": toID},
	})
	_, err := r.executeSQL(del)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	upd := Update(annotationTable).Where(And{
		Eq{annotationTable + ".item_type": r.tableName},
		Eq{annotationTable + ".item_id": fromID},
	}).Set("item_id", toID)

	c, err := r.executeSQL(upd)
	if c == 0 || errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return err
}

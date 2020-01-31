package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/model"
	"github.com/google/uuid"
)

type annotationRepository struct {
	sqlRepository
}

func NewAnnotationRepository(ctx context.Context, o orm.Ormer) model.AnnotationRepository {
	r := &annotationRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "annotation"
	return r
}

func (r *annotationRepository) upsert(values map[string]interface{}, itemType string, itemIDs ...string) error {
	upd := Update(r.tableName).Where(r.getId(itemType, itemIDs...))
	for f, v := range values {
		upd = upd.Set(f, v)
	}
	c, err := r.executeSQL(upd)
	if c == 0 || err == orm.ErrNoRows {
		for _, itemID := range itemIDs {
			id, _ := uuid.NewRandom()
			values["ann_id"] = id.String()
			values["user_id"] = userId(r.ctx)
			values["item_type"] = itemType
			values["item_id"] = itemID
			ins := Insert(r.tableName).SetMap(values)
			_, err = r.executeSQL(ins)
			if err != nil {
				return err
			}
		}
	}
	return err
}

func (r *annotationRepository) IncPlayCount(itemType, itemID string, ts time.Time) error {
	upd := Update(r.tableName).Where(r.getId(itemType, itemID)).
		Set("play_count", Expr("play_count+1")).
		Set("play_date", ts)
	c, err := r.executeSQL(upd)

	if c == 0 || err == orm.ErrNoRows {
		id, _ := uuid.NewRandom()
		values := map[string]interface{}{}
		values["ann_id"] = id.String()
		values["user_id"] = userId(r.ctx)
		values["item_type"] = itemType
		values["item_id"] = itemID
		values["play_count"] = 1
		values["play_date"] = ts
		ins := Insert(r.tableName).SetMap(values)
		_, err = r.executeSQL(ins)
		if err != nil {
			return err
		}
	}
	return err
}

func (r *annotationRepository) getId(itemType string, itemID ...string) And {
	return And{
		Eq{"user_id": userId(r.ctx)},
		Eq{"item_type": itemType},
		Eq{"item_id": itemID},
	}
}

func (r *annotationRepository) SetStar(starred bool, itemType string, ids ...string) error {
	starredAt := time.Now()
	return r.upsert(map[string]interface{}{"starred": starred, "starred_at": starredAt}, itemType, ids...)
}

func (r *annotationRepository) SetRating(rating int, itemType, itemID string) error {
	return r.upsert(map[string]interface{}{"rating": rating}, itemType, itemID)
}

func (r *annotationRepository) Delete(itemType string, itemIDs ...string) error {
	return r.delete(r.getId(itemType, itemIDs...))
}

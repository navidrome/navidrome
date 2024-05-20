package persistence

import (
	"context"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/pocketbase/dbx"
)

type tagRepository struct {
	sqlRepository
}

func NewTagRepository(ctx context.Context, db dbx.Builder) model.TagRepository {
	r := &tagRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "tag"
	return r
}

func (r *tagRepository) Add(tags ...model.Tag) error {
	return slice.RangeByChunks(tags, 200, func(chunk []model.Tag) error {
		sq := Insert(r.tableName).Columns("id", "name", "value").
			Suffix("on conflict (id) do nothing")
		for _, t := range chunk {
			sq = sq.Values(t.ID, t.Name, t.Value)
		}
		_, err := r.executeSQL(sq)
		return err
	})
}

func (r *sqlRepository) updateTags(itemID string, tags model.Tags) error {
	sqd := Delete("item_tags").Where(Eq{"item_id": itemID, "item_type": r.tableName})
	_, err := r.executeSQL(sqd)
	if err != nil {
		return err
	}
	if len(tags) == 0 {
		return nil
	}
	sqi := Insert("item_tags").Columns("item_id", "item_type", "tag_name", "tag_id").
		Suffix("on conflict (item_id, item_type, tag_id) do nothing")
	for name, values := range tags {
		for _, value := range values {
			tag := model.NewTag(name, value)
			sqi = sqi.Values(itemID, r.tableName, tag.Name, tag.ID)
		}
	}
	_, err = r.executeSQL(sqi)
	return err
}

// TODO Consolidate withTags and newSelectWithAnnotation(s)?
func (r *sqlRepository) withTags(sql SelectBuilder) SelectBuilder {
	return sql.LeftJoin("item_tags it on it.item_id = " + r.tableName + ".id and it.item_type = '" + r.tableName + "'").
		LeftJoin("tag on tag.id = it.tag_id").Columns("json_group_array(json_object(tag.name, tag.value)) as tags").
		GroupBy(r.tableName + ".id")
}

func tagIDFilter(name string, value interface{}) Sqlizer {
	tagName := strings.TrimSuffix(name, "_id")
	return Eq{"tag.id": value, "tag.name": tagName}
}

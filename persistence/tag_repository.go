package persistence

import (
	"context"

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
		sq := Insert(r.tableName).Columns("id", "tag_name", "tag_value").
			Suffix("on conflict (id) do nothing")
		for _, t := range chunk {
			sq = sq.Values(t.ID, t.TagName, t.TagValue)
		}
		_, err := r.executeSQL(sq)
		return err
	})
}

package persistence

import (
	"context"
	"fmt"
	"slices"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
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
	r.registerModel(&model.Tag{}, nil)
	return r
}

func (r *tagRepository) Add(tags ...model.Tag) error {
	for chunk := range slices.Chunk(tags, 200) {
		sq := Insert(r.tableName).Columns("id", "tag_name", "tag_value").
			Suffix("on conflict (id) do nothing")
		for _, t := range chunk {
			sq = sq.Values(t.ID, t.TagName, t.TagValue)
		}
		_, err := r.executeSQL(sq)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateCounts updates the album_count and media_file_count columns in the tag_counts table.
// Only genres are being updated for now.
func (r *tagRepository) UpdateCounts() error {
	template := `
insert into tag_counts (tag_id, %[1]s_count)
select jt.value, count(distinct (%[1]s.id))
from %[1]s
         join json_tree(tags, '$.genre') as jt
where atom is not null
  and key = 'id'
group by jt.value
on conflict (tag_id) do update
    set %[1]s_count = excluded.%[1]s_count;
`
	for _, table := range []string{"album", "media_file"} {
		query := rawSQL(fmt.Sprintf(template, table))
		_, err := r.executeSQL(query)
		if err != nil {
			return fmt.Errorf("updating %s tag counts: %w", table, err)
		}
	}
	return nil
}

func (r *tagRepository) purgeNonUsed() error {
	del := Delete(r.tableName).Where(`	not exists 
(select 1 from media_file left join json_tree(media_file.tags, '$') where atom is not null and key = 'id')
`)
	c, err := r.executeSQL(del)
	if err != nil {
		return fmt.Errorf("error purging unused tags: %w", err)
	}
	if c > 0 {
		log.Debug(r.ctx, "Purged unused tags", "totalDeleted", c)
	}
	return err
}

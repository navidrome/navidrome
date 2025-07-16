package persistence

import (
	"context"
	"fmt"
	"slices"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type tagRepository struct {
	*baseTagRepository
}

func NewTagRepository(ctx context.Context, db dbx.Builder) model.TagRepository {
	return &tagRepository{
		baseTagRepository: newBaseTagRepository(ctx, db, nil), // nil = no filter, works with all tags
	}
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
with updated_values as (
    select jt.value as id, count(distinct %[1]s.id) as %[1]s_count
    from %[1]s
             join json_tree(tags, '$.genre') as jt
    where atom is not null
      and key = 'id'
    group by jt.value
)
update tag
set %[1]s_count = updated_values.%[1]s_count
from updated_values
where tag.id = updated_values.id;
`
	for _, table := range []string{"album", "media_file"} {
		start := time.Now()
		query := Expr(fmt.Sprintf(template, table))
		c, err := r.executeSQL(query)
		log.Debug(r.ctx, "Updated tag counts", "table", table, "elapsed", time.Since(start), "updated", c)
		if err != nil {
			return fmt.Errorf("updating %s tag counts: %w", table, err)
		}
	}
	return nil
}

func (r *tagRepository) purgeUnused() error {
	del := Delete(r.tableName).Where(`	
	id not in (select jt.value
	from album left join json_tree(album.tags, '$') as jt
	where atom is not null
	  and key = 'id')
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

var _ model.ResourceRepository = &tagRepository{}

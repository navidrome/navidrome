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

func (r *tagRepository) Add(libraryID int, tags ...model.Tag) error {
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

		// Create library_tag entries for library filtering
		libSq := Insert("library_tag").Columns("tag_id", "library_id", "album_count", "media_file_count").
			Suffix("on conflict (tag_id, library_id) do nothing")
		for _, t := range chunk {
			libSq = libSq.Values(t.ID, libraryID, 0, 0)
		}
		_, err = r.executeSQL(libSq)
		if err != nil {
			return fmt.Errorf("adding library_tag entries: %w", err)
		}
	}
	return nil
}

// UpdateCounts updates the library_tag table with per-library statistics.
// Only genres are being updated for now.
func (r *tagRepository) UpdateCounts() error {
	template := `
INSERT INTO library_tag (tag_id, library_id, %[1]s_count)
SELECT jt.value as tag_id, %[1]s.library_id, count(distinct %[1]s.id) as %[1]s_count
FROM %[1]s
JOIN json_tree(%[1]s.tags, '$.genre') as jt ON jt.atom IS NOT NULL AND jt.key = 'id'
JOIN tag ON tag.id = jt.value
GROUP BY jt.value, %[1]s.library_id
ON CONFLICT (tag_id, library_id) 
DO UPDATE SET %[1]s_count = excluded.%[1]s_count;
`

	for _, table := range []string{"album", "media_file"} {
		start := time.Now()
		query := Expr(fmt.Sprintf(template, table))
		c, err := r.executeSQL(query)
		log.Debug(r.ctx, "Updated library tag counts", "table", table, "elapsed", time.Since(start), "updated", c)
		if err != nil {
			return fmt.Errorf("updating %s library tag counts: %w", table, err)
		}
	}
	return nil
}

func (r *tagRepository) purgeUnused() error {
	del := Delete(r.tableName).Where(`	
	id not in (select jt.value
	from album left join json_tree(album.tags, '$') as jt
	where atom is not null
	  and key = 'id'
	UNION 
	select jt.value
	from media_file left join json_tree(media_file.tags, '$') as jt
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

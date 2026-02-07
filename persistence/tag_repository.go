package persistence

import (
	"context"
	"fmt"
	"slices"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/db"
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
	if db.IsPostgres() {
		template = `
INSERT INTO library_tag (tag_id, library_id, %[1]s_count)
SELECT elem::text as tag_id, %[1]s.library_id, count(distinct %[1]s.id) as %[1]s_count
FROM %[1]s
CROSS JOIN LATERAL jsonb_array_elements(%[1]s.tags::jsonb->'genre') as elem
JOIN tag ON tag.id = elem::text
GROUP BY elem::text, %[1]s.library_id
ON CONFLICT (tag_id, library_id)
DO UPDATE SET %[1]s_count = excluded.%[1]s_count;
`
	}

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
	whereClause := `
	id not in (select jt.value
	from album left join json_tree(album.tags, '$') as jt
	where atom is not null
	  and key = 'id'
	UNION
	select jt.value
	from media_file left join json_tree(media_file.tags, '$') as jt
	where atom is not null
	  and key = 'id')
`
	if db.IsPostgres() {
		whereClause = `
	id not in (
		SELECT elem->>'id'
		FROM album, LATERAL jsonb_each(album.tags::jsonb) as tag_type,
		     LATERAL jsonb_array_elements(tag_type.value) as elem
		WHERE elem->>'id' IS NOT NULL
		UNION
		SELECT elem->>'id'
		FROM media_file, LATERAL jsonb_each(media_file.tags::jsonb) as tag_type,
		     LATERAL jsonb_array_elements(tag_type.value) as elem
		WHERE elem->>'id' IS NOT NULL
	)
`
	}
	del := Delete(r.tableName).Where(whereClause)
	c, err := r.executeSQL(del)
	if err != nil {
		return fmt.Errorf("error purging %s unused tags: %w", r.tableName, err)
	}
	if c > 0 {
		log.Debug(r.ctx, "Purged unused tags", "totalDeleted", c, "table", r.tableName)
	}
	return err
}

var _ model.ResourceRepository = &tagRepository{}

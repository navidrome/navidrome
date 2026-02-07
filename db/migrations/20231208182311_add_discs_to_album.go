package migrations

import (
	"context"
	"database/sql"

	"github.com/navidrome/navidrome/db/dialect"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddDiscToAlbum, downAddDiscToAlbum)
}

func upAddDiscToAlbum(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `alter table album add discs JSONB default '{}';`)
	if err != nil {
		return err
	}

	updateQuery := `
update album set discs = t.discs
from (select album_id, json_group_object(disc_number, disc_subtitle) as discs
      from (select distinct album_id, disc_number, disc_subtitle
            from media_file
            where disc_number > 0
            order by album_id, disc_number)
      group by album_id
      having discs <> '{"1":""}') as t
where album.id = t.album_id;
`
	if dialect.Current != nil && dialect.Current.Name() == "postgres" {
		updateQuery = `
update album set discs = t.discs
from (select album_id, jsonb_object_agg(disc_number::text, disc_subtitle) as discs
      from (select distinct album_id, disc_number, disc_subtitle
            from media_file
            where disc_number > 0
            order by album_id, disc_number) as subq
      group by album_id
      having jsonb_object_agg(disc_number::text, disc_subtitle) <> '{"1":""}') as t
where album.id = t.album_id;
`
	}
	_, err = tx.ExecContext(ctx, updateQuery)
	return err
}

func downAddDiscToAlbum(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `alter table album drop discs;`)
	return err
}

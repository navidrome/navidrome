package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20200325185135, Down20200325185135)
}

func Up20200325185135(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
alter table album
	add album_artist_id varchar(255) default '';
create index album_artist_album_id
	on album (album_artist_id);

alter table media_file
	add album_artist_id varchar(255) default '';
create index media_file_artist_album_id
	on media_file (album_artist_id);
`)
	if err != nil {
		return err
	}
	notice(ctx, tx, "A full rescan will be performed!")
	return forceFullRescan(ctx, tx)
}

func Down20200325185135(_ context.Context, _ *sql.Tx) error {
	return nil
}

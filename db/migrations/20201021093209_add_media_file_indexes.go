package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20201021093209, Down20201021093209)
}

func Up20201021093209(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
create index if not exists media_file_artist
	on media_file (artist);
create index if not exists media_file_album_artist
	on media_file (album_artist);
create index if not exists media_file_mbz_track_id
	on media_file (mbz_track_id);
`)
	return err
}

func Down20201021093209(_ context.Context, tx *sql.Tx) error {
	return nil
}

package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upFixTotalSongs, downFixTotalSongs)
}

func upFixTotalSongs(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
update library set total_songs = (
    select count(*) from media_file where library_id = library.id and missing = 0
);
`)
	return err
}

func downFixTotalSongs(ctx context.Context, tx *sql.Tx) error {
	return nil
}

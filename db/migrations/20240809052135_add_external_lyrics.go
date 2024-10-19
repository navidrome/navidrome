package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddExternalLyrics, downAddExternalLyrics)
}

func upAddExternalLyrics(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
alter table media_file
	add external_lyrics text default '' not null;
alter table media_file
	add external_lyrics_updated_at datetime;
	`)

	return err
}

func downAddExternalLyrics(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}

package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddPlaylistPath, downAddPlaylistPath)
}

func upAddPlaylistPath(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table playlist
	add path string default '' not null;

alter table playlist
	add sync bool default false not null;
`)

	return err
}

func downAddPlaylistPath(_ context.Context, tx *sql.Tx) error {
	return nil
}

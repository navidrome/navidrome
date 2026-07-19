package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddAlbumReplaygain, downAddAlbumReplaygain)
}

func upAddAlbumReplaygain(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
ALTER TABLE album ADD COLUMN rg_album_gain real;
ALTER TABLE album ADD COLUMN rg_album_peak real;

UPDATE album SET
	rg_album_gain = (SELECT max(rg_album_gain) FROM media_file WHERE media_file.album_id = album.id),
	rg_album_peak = (SELECT max(rg_album_peak) FROM media_file WHERE media_file.album_id = album.id);
	`)
	return err
}

func downAddAlbumReplaygain(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}

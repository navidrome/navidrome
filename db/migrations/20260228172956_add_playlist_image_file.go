package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddPlaylistImageFile, downAddPlaylistImageFile)
}

func upAddPlaylistImageFile(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `ALTER TABLE playlist ADD COLUMN image_file VARCHAR(255) DEFAULT '';`)
	return err
}

func downAddPlaylistImageFile(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `ALTER TABLE playlist DROP COLUMN image_file;`)
	return err
}

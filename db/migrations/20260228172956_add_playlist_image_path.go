package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddPlaylistImagePath, downAddPlaylistImagePath)
}

func upAddPlaylistImagePath(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `ALTER TABLE playlist ADD COLUMN image_path VARCHAR(255) DEFAULT '';`)
	return err
}

func downAddPlaylistImagePath(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `ALTER TABLE playlist DROP COLUMN image_path;`)
	return err
}

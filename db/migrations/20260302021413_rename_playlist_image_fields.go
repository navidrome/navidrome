package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upRenamePlaylistImageFields, downRenamePlaylistImageFields)
}

func upRenamePlaylistImageFields(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `ALTER TABLE playlist RENAME COLUMN image_file TO uploaded_image;`)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `ALTER TABLE playlist ADD COLUMN external_image_url VARCHAR(255) DEFAULT '';`)
	return err
}

func downRenamePlaylistImageFields(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `ALTER TABLE playlist DROP COLUMN external_image_url;`)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `ALTER TABLE playlist RENAME COLUMN uploaded_image TO image_file;`)
	return err
}

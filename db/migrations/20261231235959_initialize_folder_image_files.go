package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upInitializeFolderImageFiles, downInitializeFolderImageFiles)
}

func upInitializeFolderImageFiles(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `UPDATE folder SET image_files = '[]' WHERE image_files IS NULL OR image_files = '';`)
	return err
}

func downInitializeFolderImageFiles(ctx context.Context, tx *sql.Tx) error {
	return nil
}

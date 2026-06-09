package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddRadioUploadedImage, downAddRadioUploadedImage)
}

func upAddRadioUploadedImage(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `ALTER TABLE radio ADD COLUMN uploaded_image VARCHAR(255) NOT NULL DEFAULT ''`)
	return err
}

func downAddRadioUploadedImage(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}

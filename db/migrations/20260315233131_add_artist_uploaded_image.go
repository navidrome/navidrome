package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddArtistUploadedImage, downAddArtistUploadedImage)
}

func upAddArtistUploadedImage(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `ALTER TABLE artist ADD COLUMN uploaded_image VARCHAR(255) DEFAULT ''`)
	return err
}

func downAddArtistUploadedImage(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}

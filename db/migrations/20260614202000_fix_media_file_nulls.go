package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upFixAllMediaFileNulls, downFixAllMediaFileNulls)
}

func upFixAllMediaFileNulls(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "UPDATE media_file SET bpm = 0 WHERE bpm IS NULL; UPDATE media_file SET bit_depth = 0 WHERE bit_depth IS NULL; UPDATE media_file SET sample_rate = 0 WHERE sample_rate IS NULL;")
	return err
}

func downFixAllMediaFileNulls(ctx context.Context, tx *sql.Tx) error {
	return nil
}

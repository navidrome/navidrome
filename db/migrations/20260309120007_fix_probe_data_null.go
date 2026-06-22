package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upFixProbeDataNull, downFixProbeDataNull)
}

func upFixProbeDataNull(ctx context.Context, tx *sql.Tx) error {
	// Recreate probe_data column as NOT NULL with empty string default.
	// The previous migration created it with DEFAULT NULL, which causes
	// scan errors when reading into Go string fields.
	_, err := tx.ExecContext(ctx, `ALTER TABLE media_file DROP COLUMN probe_data`)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `ALTER TABLE media_file ADD COLUMN probe_data TEXT DEFAULT '' NOT NULL`)
	return err
}

func downFixProbeDataNull(_ context.Context, _ *sql.Tx) error {
	return nil
}

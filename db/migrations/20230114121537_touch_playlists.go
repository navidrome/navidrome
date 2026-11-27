package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upTouchPlaylists, downTouchPlaylists)
}

func upTouchPlaylists(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `update playlist set updated_at = datetime('now');`)
	return err
}

func downTouchPlaylists(_ context.Context, _ *sql.Tx) error {
	return nil
}

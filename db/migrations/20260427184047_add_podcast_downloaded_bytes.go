package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddPodcastDownloadedBytes, downAddPodcastDownloadedBytes)
}

func upAddPodcastDownloadedBytes(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `ALTER TABLE podcast_episode ADD COLUMN downloaded_bytes INTEGER NOT NULL DEFAULT 0`)
	return err
}

func downAddPodcastDownloadedBytes(ctx context.Context, tx *sql.Tx) error {
	return nil
}

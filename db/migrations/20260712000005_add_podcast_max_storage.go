package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddPodcastMaxStorage, downAddPodcastMaxStorage)
}

func upAddPodcastMaxStorage(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `alter table podcast_channel add column max_storage_mb integer default 0 not null;`)
	return err
}

func downAddPodcastMaxStorage(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `alter table podcast_channel drop column max_storage_mb;`)
	return err
}

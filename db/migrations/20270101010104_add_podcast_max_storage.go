package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddPodcastMaxStorage, downAddPodcastMaxStorage)
}

func upAddPodcastMaxStorage(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`alter table podcast_channel add column max_storage_mb integer default 0 not null;`)
	return err
}

func downAddPodcastMaxStorage(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`alter table podcast_channel drop column max_storage_mb;`)
	return err
}

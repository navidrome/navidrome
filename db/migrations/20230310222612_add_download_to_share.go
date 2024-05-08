package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddDownloadToShare, downAddDownloadToShare)
}

func upAddDownloadToShare(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table share
	add downloadable bool not null default false;
`)
	return err
}

func downAddDownloadToShare(_ context.Context, tx *sql.Tx) error {
	return nil
}

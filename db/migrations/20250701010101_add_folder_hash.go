package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddFolderHash, downAddFolderHash)
}

func upAddFolderHash(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `alter table folder add column hash varchar default '' not null;`)
	return err
}

func downAddFolderHash(ctx context.Context, tx *sql.Tx) error {
	return nil
}

package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddSyncPlayqueueColumnToUserTable, downAddSyncPlayqueueColumnToUserTable)
}

func upAddSyncPlayqueueColumnToUserTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
	ALTER TABLE "user" ADD sync_playqueue BOOL DEFAULT (FALSE) NOT NULL;
	`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan needs to be performed to import more tags")
	return forceFullRescan(tx)
}

func downAddSyncPlayqueueColumnToUserTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
	ALTER TABLE "user"  DROP COLUMN "sync_playqueue";
	`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan needs to be performed to import more tags")
	return forceFullRescan(tx)
}

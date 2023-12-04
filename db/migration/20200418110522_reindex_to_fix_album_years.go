package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20200418110522, Down20200418110522)
}

func Up20200418110522(_ context.Context, tx *sql.Tx) error {
	notice(tx, "A full rescan will be performed to fix search Albums by year")
	return forceFullRescan(tx)
}

func Down20200418110522(_ context.Context, tx *sql.Tx) error {
	return nil
}

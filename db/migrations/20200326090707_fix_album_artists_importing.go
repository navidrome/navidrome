package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20200326090707, Down20200326090707)
}

func Up20200326090707(_ context.Context, tx *sql.Tx) error {
	notice(tx, "A full rescan will be performed!")
	return forceFullRescan(tx)
}

func Down20200326090707(_ context.Context, tx *sql.Tx) error {
	return nil
}

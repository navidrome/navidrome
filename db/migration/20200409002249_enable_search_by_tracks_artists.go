package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20200409002249, Down20200409002249)
}

func Up20200409002249(_ context.Context, tx *sql.Tx) error {
	notice(tx, "A full rescan will be performed to enable search by individual Artist in an Album!")
	return forceFullRescan(tx)
}

func Down20200409002249(_ context.Context, tx *sql.Tx) error {
	return nil
}

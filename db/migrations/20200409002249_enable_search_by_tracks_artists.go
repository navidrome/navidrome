package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20200409002249, Down20200409002249)
}

func Up20200409002249(ctx context.Context, tx *sql.Tx) error {
	notice(ctx, tx, "A full rescan will be performed to enable search by individual Artist in an Album!")
	return forceFullRescan(ctx, tx)
}

func Down20200409002249(_ context.Context, _ *sql.Tx) error {
	return nil
}

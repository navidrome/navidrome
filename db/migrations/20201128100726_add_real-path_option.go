package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20201128100726, Down20201128100726)
}

func Up20201128100726(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table player
	add report_real_path bool default FALSE not null;
`)
	return err
}

func Down20201128100726(_ context.Context, tx *sql.Tx) error {
	return nil
}

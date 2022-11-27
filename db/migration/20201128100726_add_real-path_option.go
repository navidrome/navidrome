package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20201128100726, Down20201128100726)
}

func Up20201128100726(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table player
	add report_real_path bool default FALSE not null;
`)
	return err
}

func Down20201128100726(tx *sql.Tx) error {
	return nil
}

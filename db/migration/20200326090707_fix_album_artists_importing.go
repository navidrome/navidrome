package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(Up20200326090707, Down20200326090707)
}

func Up20200326090707(tx *sql.Tx) error {
	notice(tx, "A full rescan will be performed!")
	return forceFullRescan(tx)
}

func Down20200326090707(tx *sql.Tx) error {
	return nil
}

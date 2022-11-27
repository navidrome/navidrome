package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200409002249, Down20200409002249)
}

func Up20200409002249(tx *sql.Tx) error {
	notice(tx, "A full rescan will be performed to enable search by individual Artist in an Album!")
	return forceFullRescan(tx)
}

func Down20200409002249(tx *sql.Tx) error {
	return nil
}

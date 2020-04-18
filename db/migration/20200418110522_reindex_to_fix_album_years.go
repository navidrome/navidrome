package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200418110522, Down20200418110522)
}

func Up20200418110522(tx *sql.Tx) error {
	notice(tx, "A full rescan will be performed to fix search Albums by year")
	return forceFullRescan(tx)
}

func Down20200418110522(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}

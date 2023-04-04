package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(Up20200419222708, Down20200419222708)
}

func Up20200419222708(tx *sql.Tx) error {
	notice(tx, "A full rescan will be performed to change the search behaviour")
	return forceFullRescan(tx)
}

func Down20200419222708(tx *sql.Tx) error {
	return nil
}

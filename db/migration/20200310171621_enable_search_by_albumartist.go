package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200310171621, Down20200310171621)
}

func Up20200310171621(tx *sql.Tx) error {
	notice(tx, "A full rescan will be performed to enable search by Album Artist!")
	return forceFullRescan(tx)
}

func Down20200310171621(tx *sql.Tx) error {
	return nil
}

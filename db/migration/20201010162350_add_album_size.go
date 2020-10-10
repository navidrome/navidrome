package migration

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20201010162350, Down20201010162350)
}

func Up20201010162350(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table album
	add size integer default 0 not null;
	`)
	return err
	/*  IS A RESCAN NECESSARY TO UPDATE ALBUM TABLE WITH SIZES?
	if err != nil {
		return err
	}
	notice(tx, "A full rescan will be performed to calculate album sizes.")
	return forceFullRescan(tx)
	*/
}

func Down20201010162350(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}

package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(Up20231116000001, Down20231116000001)
}

func Up20231116000001(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file
	add classical bool;
alter table media_file
    add work_title varchar(255) default '' not null;
alter table album
	add classical bool;
`)
	if err != nil {
		return err
	}

	notice(tx, "A full rescan needs to be performed to detect Classical")
	return forceFullRescan(tx)
}

func Down20231116000001(tx *sql.Tx) error {
	return nil
}

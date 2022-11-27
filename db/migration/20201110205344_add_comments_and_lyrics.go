package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20201110205344, Down20201110205344)
}

func Up20201110205344(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file
	add comment varchar;
alter table media_file
	add lyrics varchar;

alter table album
	add comment varchar;
`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan will be performed to import comments and lyrics")
	return forceFullRescan(tx)
}

func Down20201110205344(tx *sql.Tx) error {
	return nil
}

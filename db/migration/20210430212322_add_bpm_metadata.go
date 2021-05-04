package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(upAddBpmMetadata, downAddBpmMetadata)
}

func upAddBpmMetadata(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file
add bpm integer;
`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan needs to be performed to import more tags")
	return forceFullRescan(tx)
}

func downAddBpmMetadata(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file
	drop bpm;
`)
	return err
}

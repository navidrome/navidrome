package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upAddBpmMetadata, downAddBpmMetadata)
}

func upAddBpmMetadata(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file
    add bpm integer;

create index if not exists media_file_bpm
	on media_file (bpm);
`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan needs to be performed to import more tags")
	return forceFullRescan(tx)
}

func downAddBpmMetadata(tx *sql.Tx) error {
	return nil
}

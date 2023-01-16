package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(upAddMediafileChannels, downAddMediafileChannels)
}

func upAddMediafileChannels(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file
    add release_year integer;
    add recording_year integer;
alter table album
    add min_release_year integer;
    add max_release_year integer;
    add min_recording_year integer;
    add max_recording_year integer;
`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan needs to be performed to import more tags")
	return forceFullRescan(tx)
}

func downAddMediafileChannels(tx *sql.Tx) error {
	return nil
}

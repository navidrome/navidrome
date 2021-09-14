package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(UpAddMetadata, DownAddMetadata)
}

func UpAddMetadata(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file
	add song_subtitle varchar(255);
alter table media_file
	add work varchar(255);
alter table media_file
	add movement_number integer not null default 0;
alter table media_file
	add movement_name varchar(255);
`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan needs to be performed to import more tags")
	return forceFullRescan(tx)
}

func DownAddMetadata(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}

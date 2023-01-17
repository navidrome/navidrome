package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(upAddReplaygainMetadata, downAddReplaygainMetadata)
}

func upAddReplaygainMetadata(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file add 
	rg_album_gain real;
alter table media_file add 
	rg_album_peak real;
alter table media_file add 
	rg_track_gain real;
alter table media_file add 
	rg_track_peak real;
`)
	if err != nil {
		return err
	}

	notice(tx, "A full rescan needs to be performed to import more tags")
	return forceFullRescan(tx)
}

func downAddReplaygainMetadata(tx *sql.Tx) error {
	return nil
}

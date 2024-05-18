package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddReplaygainMetadata, downAddReplaygainMetadata)
}

func upAddReplaygainMetadata(_ context.Context, tx *sql.Tx) error {
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

func downAddReplaygainMetadata(_ context.Context, tx *sql.Tx) error {
	return nil
}

package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(upAddMusicbrainzReleaseTrackId, downAddMusicbrainzReleaseTrackId)
}

func upAddMusicbrainzReleaseTrackId(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file
	add mbz_release_track_id varchar(255);
`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan needs to be performed to import more tags")
	return forceFullRescan(tx)
}

func downAddMusicbrainzReleaseTrackId(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}

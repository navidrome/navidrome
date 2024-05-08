package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddMusicbrainzReleaseTrackId, downAddMusicbrainzReleaseTrackId)
}

func upAddMusicbrainzReleaseTrackId(_ context.Context, tx *sql.Tx) error {
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

func downAddMusicbrainzReleaseTrackId(_ context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}

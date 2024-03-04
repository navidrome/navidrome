package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddMultiTrackMediaSupport, downAddMultiTrackMediaSupport)
}

func upAddMultiTrackMediaSupport(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file add
    offset real default 0 not null;
	
alter table media_file add
    sub_track integer default -1 not null;

create unique index if not exists media_file_path_with_sub_track
	on media_file (path, sub_track);
`)
	if err != nil {
		return err
	}

	notice(tx, "A full rescan needs to be performed to import more tags")
	return forceFullRescan(tx)
}

func downAddMultiTrackMediaSupport(_ context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}

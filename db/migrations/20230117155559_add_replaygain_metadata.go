package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddReplaygainMetadata, downAddReplaygainMetadata)
}

func upAddReplaygainMetadata(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
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

	notice(ctx, tx, "A full rescan needs to be performed to import more tags")
	return forceFullRescan(ctx, tx)
}

func downAddReplaygainMetadata(_ context.Context, _ *sql.Tx) error {
	return nil
}

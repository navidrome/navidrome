package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddBpmMetadata, downAddBpmMetadata)
}

func upAddBpmMetadata(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
alter table media_file
    add bpm integer;

create index if not exists media_file_bpm
	on media_file (bpm);
`)
	if err != nil {
		return err
	}
	notice(ctx, tx, "A full rescan needs to be performed to import more tags")
	return forceFullRescan(ctx, tx)
}

func downAddBpmMetadata(_ context.Context, _ *sql.Tx) error {
	return nil
}

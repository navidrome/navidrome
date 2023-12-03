package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddMediafileChannels, downAddMediafileChannels)
}

func upAddMediafileChannels(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file
    add channels integer;

create index if not exists media_file_channels
	on media_file (channels);
`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan needs to be performed to import more tags")
	return forceFullRescan(tx)
}

func downAddMediafileChannels(_ context.Context, tx *sql.Tx) error {
	return nil
}

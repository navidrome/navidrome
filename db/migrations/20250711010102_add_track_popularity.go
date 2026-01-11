package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddTrackPopularity, downAddTrackPopularity)
}

func upAddTrackPopularity(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file
	add lastfm_listeners integer default 0 not null;
alter table media_file
	add lastfm_playcount integer default 0 not null;
`)
	return err
}

func downAddTrackPopularity(_ context.Context, tx *sql.Tx) error {
	return nil
}

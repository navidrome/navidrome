package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddPlaylistExternalInfo, downAddPlaylistExternalInfo)
}

func upAddPlaylistExternalInfo(_ context.Context, tx *sql.Tx) error {
	// Note: Ideally, we would also change the type of "comment" to be longer than 255
	// characters, but since this is Sqlite, the length doesn't matter
	_, err := tx.Exec(`
alter table playlist
	add external_agent varchar default '' not null;
alter table playlist 
	add external_id varchar default '' not null;
alter table playlist
	add external_url varchar default '' not null;
alter table playlist
	add external_sync bool default false;
alter table playlist
	add external_syncable bool default false;
alter table playlist
	add external_recommended bool default false;
	`)
	return err
}

func downAddPlaylistExternalInfo(_ context.Context, tx *sql.Tx) error {
	return nil
}

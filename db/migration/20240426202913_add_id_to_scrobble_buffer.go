package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddIdToScrobbleBuffer, downAddIdToScrobbleBuffer)
}

func upAddIdToScrobbleBuffer(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
delete from scrobble_buffer where user_id <> '';
alter table scrobble_buffer add id varchar not null default '';
create unique index scrobble_buffer_id_ix
    on scrobble_buffer (id);
`)
	return err
}

func downAddIdToScrobbleBuffer(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
drop index scrobble_buffer_id_ix;
alter table scrobble_buffer drop id;
`)
	return err
}

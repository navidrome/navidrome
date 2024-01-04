package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddScrobbleBuffer, downAddScrobbleBuffer)
}

func upAddScrobbleBuffer(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
create table if not exists scrobble_buffer
(
	user_id varchar not null
	constraint scrobble_buffer_user_id_fk
		references "user"
			on update cascade on delete cascade,
	service varchar not null,
	media_file_id varchar not null
		constraint scrobble_buffer_media_file_id_fk
			references media_file
				on update cascade on delete cascade,
	play_time timestamp not null,
	enqueue_time timestamp not null default current_timestamp,
	constraint scrobble_buffer_pk
		unique (user_id, service, media_file_id, play_time)
);
`)

	return err
}

func downAddScrobbleBuffer(_ context.Context, tx *sql.Tx) error {
	return nil
}

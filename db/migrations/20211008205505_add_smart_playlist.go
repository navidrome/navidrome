package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddSmartPlaylist, downAddSmartPlaylist)
}

func upAddSmartPlaylist(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table playlist
	add column rules varchar null;
alter table playlist
	add column evaluated_at datetime null;
create index if not exists playlist_evaluated_at
	on playlist(evaluated_at);

create table playlist_fields (
    field varchar(255) not null, 
	playlist_id varchar(255) not null
		constraint playlist_fields_playlist_id_fk
			references playlist
				on update cascade on delete cascade
);
create unique index playlist_fields_idx
	on playlist_fields (field, playlist_id);
`)
	return err
}

func downAddSmartPlaylist(_ context.Context, tx *sql.Tx) error {
	return nil
}

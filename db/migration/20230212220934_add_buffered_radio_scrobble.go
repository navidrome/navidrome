package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(upAddRadioScrobble, downAddRadioScrobble)
}

func upAddRadioScrobble(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec(`
create table if not exists scrobble_radio
(
	user_id varchar not null
	constraint scrobble_radio_user_id_fk
		references user
			on update cascade on delete cascade,
	service varchar not null,
	play_time datetime not null,
	enqueue_time datetime not null default current_timestamp,
	artist varchar not null,
	title varchar not null,
	constraint scrobble_radio_pk
		unique (user_id, service, play_time, artist, title)
);
	`)

	return err
}

func downAddRadioScrobble(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}

package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(upAddStarBuffer, downAddStarBuffer)
}

func upAddStarBuffer(tx *sql.Tx) error {
	_, err := tx.Exec(`
create table if not exists star_buffer
(
	user_id varchar not null
	constraint star_buffer_user_id_fk
		references user
			on update cascade on delete cascade,
	service varchar not null,
	media_file_id varchar not null
		constraint star_buffer_media_file_id_fk
			references media_file
				on update cascade on delete cascade,
	is_star bool not null,
	enqueue_time datetime not null default current_timestamp,
	constraint star_buffer_pk
		unique (user_id, service, media_file_id, user_id)
);
`)

	return err
}

func downAddStarBuffer(tx *sql.Tx) error {
	return nil
}

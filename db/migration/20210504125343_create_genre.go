package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20210430125343, Down20210430125343)
}

func Up20210430125343(tx *sql.Tx) error {
	_, err := tx.Exec(`
create table if not exists genre
(		
	name varchar(255) not null
		primary key,
	song_count integer default 0 not null,
	album_count integer default 0 not null
);

create table if not exists genre_type
(
	genre_id varchar(255) not null
		references genre
			on update cascade on delete cascade,
	item_id varchar(255) not null,
	item_type varchar(255) not null,
	unique (genre_id, item_id, item_type)
);
	`)

	return err
}

func Down20210430125343(tx *sql.Tx) error {
	return nil
}

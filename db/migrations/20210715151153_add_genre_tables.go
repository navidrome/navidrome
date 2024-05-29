package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddGenreTables, downAddGenreTables)
}

func upAddGenreTables(_ context.Context, tx *sql.Tx) error {
	notice(tx, "A full rescan will be performed to import multiple genres!")
	_, err := tx.Exec(`
create table if not exists genre
(
  id varchar not null primary key,
  name varchar not null,  
	constraint genre_name_ux
		unique (name)
);

create table if not exists  album_genres
(
	album_id varchar default null not null
		references album
			on delete cascade,
	genre_id varchar default null not null
		references genre
			on delete cascade,
	constraint album_genre_ux
		unique (album_id, genre_id)
);

create table if not exists  media_file_genres
(
	media_file_id varchar default null not null
		references media_file
			on delete cascade,
	genre_id varchar default null not null
		references genre
			on delete cascade,
	constraint media_file_genre_ux
		unique (media_file_id, genre_id)
);

create table if not exists  artist_genres
(
	artist_id varchar default null not null
		references artist
			on delete cascade,
	genre_id varchar default null not null
		references genre
			on delete cascade,
	constraint artist_genre_ux
		unique (artist_id, genre_id)
);
`)
	if err != nil {
		return err
	}
	return forceFullRescan(tx)
}

func downAddGenreTables(_ context.Context, tx *sql.Tx) error {
	return nil
}

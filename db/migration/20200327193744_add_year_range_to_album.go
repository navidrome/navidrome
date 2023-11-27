package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20200327193744, Down20200327193744)
}

func Up20200327193744(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
create table album_dg_tmp
(
	id varchar(255) not null
		primary key,
	name varchar(255) default '' not null,
	artist_id varchar(255) default '' not null,
	cover_art_path varchar(255) default '' not null,
	cover_art_id varchar(255) default '' not null,
	artist varchar(255) default '' not null,
	album_artist varchar(255) default '' not null,
    min_year int default 0 not null,
	max_year integer default 0 not null,
	compilation bool default FALSE not null,
	song_count integer default 0 not null,
	duration real default 0 not null,
	genre varchar(255) default '' not null,
	created_at datetime,
	updated_at datetime,
	full_text varchar(255) default '',
	album_artist_id varchar(255) default ''
);

insert into album_dg_tmp(id, name, artist_id, cover_art_path, cover_art_id, artist, album_artist, max_year, compilation, song_count, duration, genre, created_at, updated_at, full_text, album_artist_id) select id, name, artist_id, cover_art_path, cover_art_id, artist, album_artist, year, compilation, song_count, duration, genre, created_at, updated_at, full_text, album_artist_id from album;

drop table album;

alter table album_dg_tmp rename to album;

create index album_artist
	on album (artist);

create index album_artist_album
	on album (artist);

create index album_artist_album_id
	on album (album_artist_id);

create index album_artist_id
	on album (artist_id);

create index album_full_text
	on album (full_text);

create index album_genre
	on album (genre);

create index album_name
	on album (name);

create index album_min_year
	on album (min_year);

create index album_max_year
	on album (max_year);

`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan will be performed!")
	return forceFullRescan(tx)
}

func Down20200327193744(_ context.Context, tx *sql.Tx) error {
	return nil
}

package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20200220143731, Down20200220143731)
}

func Up20200220143731(_ context.Context, tx *sql.Tx) error {
	notice(tx, "This migration will force the next scan to be a full rescan!")
	_, err := tx.Exec(`
create table media_file_dg_tmp
(
	id varchar(255) not null
		primary key,
	path varchar(255) default '' not null,
	title varchar(255) default '' not null,
	album varchar(255) default '' not null,
	artist varchar(255) default '' not null,
	artist_id varchar(255) default '' not null,
	album_artist varchar(255) default '' not null,
	album_id varchar(255) default '' not null,
	has_cover_art bool default FALSE not null,
	track_number integer default 0 not null,
	disc_number integer default 0 not null,
	year integer default 0 not null,
	size integer default 0 not null,
	suffix varchar(255) default '' not null,
	duration real default 0 not null,
	bit_rate integer default 0 not null,
	genre varchar(255) default '' not null,
	compilation bool default FALSE not null,
	created_at datetime,
	updated_at datetime
);

insert into media_file_dg_tmp(id, path, title, album, artist, artist_id, album_artist, album_id, has_cover_art, track_number, disc_number, year, size, suffix, duration, bit_rate, genre, compilation, created_at, updated_at) select id, path, title, album, artist, artist_id, album_artist, album_id, has_cover_art, track_number, disc_number, year, size, suffix, duration, bit_rate, genre, compilation, created_at, updated_at from media_file;

drop table media_file;

alter table media_file_dg_tmp rename to media_file;

create index media_file_album_id
	on media_file (album_id);

create index media_file_genre
	on media_file (genre);

create index media_file_path
	on media_file (path);

create index media_file_title
	on media_file (title);

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
	year integer default 0 not null,
	compilation bool default FALSE not null,
	song_count integer default 0 not null,
	duration real default 0 not null,
	genre varchar(255) default '' not null,
	created_at datetime,
	updated_at datetime
);

insert into album_dg_tmp(id, name, artist_id, cover_art_path, cover_art_id, artist, album_artist, year, compilation, song_count, duration, genre, created_at, updated_at) select id, name, artist_id, cover_art_path, cover_art_id, artist, album_artist, year, compilation, song_count, duration, genre, created_at, updated_at from album;

drop table album;

alter table album_dg_tmp rename to album;

create index album_artist
	on album (artist);

create index album_artist_id
	on album (artist_id);

create index album_genre
	on album (genre);

create index album_name
	on album (name);

create index album_year
	on album (year);

create table playlist_dg_tmp
(
	id varchar(255) not null
		primary key,
	name varchar(255) default '' not null,
	comment varchar(255) default '' not null,
	duration real default 0 not null,
	owner varchar(255) default '' not null,
	public bool default FALSE not null,
	tracks text not null
);

insert into playlist_dg_tmp(id, name, comment, duration, owner, public, tracks) select id, name, comment, duration, owner, public, tracks from playlist;

drop table playlist;

alter table playlist_dg_tmp rename to playlist;

create index playlist_name
	on playlist (name);

-- Force a full rescan
delete from property where id like 'LastScan%';
update media_file set updated_at = '0001-01-01';
`)
	return err
}

func Down20200220143731(_ context.Context, tx *sql.Tx) error {
	return nil
}

package migrations

import (
	"context"
	"database/sql"

	"github.com/navidrome/navidrome/log"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20200130083147, Down20200130083147)
}

func Up20200130083147(_ context.Context, tx *sql.Tx) error {
	log.Info("Creating DB Schema")
	_, err := tx.Exec(`
create table if not exists album
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
	duration integer default 0 not null,
	genre varchar(255) default '' not null,
	created_at datetime,
	updated_at datetime
);

create index if not exists album_artist
	on album (artist);

create index if not exists album_artist_id
	on album (artist_id);

create index if not exists album_genre
	on album (genre);

create index if not exists album_name
	on album (name);

create index if not exists album_year
	on album (year);

create table if not exists annotation
(
	ann_id varchar(255) not null
		primary key,
	user_id varchar(255) default '' not null,
	item_id varchar(255) default '' not null,
	item_type varchar(255) default '' not null,
	play_count integer,
	play_date datetime,
	rating integer,
	starred bool default FALSE not null,
	starred_at datetime,
	unique (user_id, item_id, item_type)
);

create index if not exists annotation_play_count
	on annotation (play_count);

create index if not exists annotation_play_date
	on annotation (play_date);

create index if not exists annotation_rating
	on annotation (rating);

create index if not exists annotation_starred
	on annotation (starred);

create table if not exists artist
(
	id varchar(255) not null
		primary key,
	name varchar(255) default '' not null,
	album_count integer default 0 not null
);

create index if not exists artist_name
	on artist (name);

create table if not exists media_file
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
	duration integer default 0 not null,
	bit_rate integer default 0 not null,
	genre varchar(255) default '' not null,
	compilation bool default FALSE not null,
	created_at datetime,
	updated_at datetime
);

create index if not exists media_file_album_id
	on media_file (album_id);

create index if not exists media_file_genre
	on media_file (genre);

create index if not exists media_file_path
	on media_file (path);

create index if not exists media_file_title
	on media_file (title);

create table if not exists playlist
(
	id varchar(255) not null
		primary key,
	name varchar(255) default '' not null,
	comment varchar(255) default '' not null,
	duration integer default 0 not null,
	owner varchar(255) default '' not null,
	public bool default FALSE not null,
	tracks text not null
);

create index if not exists playlist_name
	on playlist (name);

create table if not exists property
(
	id varchar(255) not null
		primary key,
	value varchar(255) default '' not null
);

create table if not exists search
(
	id varchar(255) not null
		primary key,
	"table" varchar(255) default '' not null,
	full_text varchar(255) default '' not null
);

create index if not exists search_full_text
	on search (full_text);

create index if not exists search_table
	on search ("table");

create table if not exists user
(
	id varchar(255) not null
		primary key,
	user_name varchar(255) default '' not null
		unique,
	name varchar(255) default '' not null,
	email varchar(255) default '' not null
		unique,
	password varchar(255) default '' not null,
	is_admin bool default FALSE not null,
	last_login_at datetime,
	last_access_at datetime,
	created_at datetime not null,
	updated_at datetime not null
);`)
	return err
}

func Down20200130083147(_ context.Context, tx *sql.Tx) error {
	return nil
}

package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20200608153717, Down20200608153717)
}

func Up20200608153717(_ context.Context, tx *sql.Tx) error {
	// First delete dangling players
	_, err := tx.Exec(`
delete from player where user_name not in (select user_name from user)`)
	if err != nil {
		return err
	}

	// Also delete dangling players
	_, err = tx.Exec(`
delete from playlist where owner not in (select user_name from user)`)
	if err != nil {
		return err
	}

	// Also delete dangling playlist tracks
	_, err = tx.Exec(`
delete from playlist_tracks where playlist_id not in (select id from playlist)`)
	if err != nil {
		return err
	}

	// Add foreign key to player table
	err = updatePlayer_20200608153717(tx)
	if err != nil {
		return err
	}

	// Add foreign key to playlist table
	err = updatePlaylist_20200608153717(tx)
	if err != nil {
		return err
	}

	// Add foreign keys to playlist_tracks table
	return updatePlaylistTracks_20200608153717(tx)
}

func updatePlayer_20200608153717(tx *sql.Tx) error {
	_, err := tx.Exec(`
create table player_dg_tmp
(
	id varchar(255) not null
		primary key,
	name varchar not null
		unique,
	type varchar,
	user_name varchar not null
		references user (user_name)
			on update cascade on delete cascade,
	client varchar not null,
	ip_address varchar,
	last_seen timestamp,
	max_bit_rate int default 0,
	transcoding_id varchar null
);

insert into player_dg_tmp(id, name, type, user_name, client, ip_address, last_seen, max_bit_rate, transcoding_id) select id, name, type, user_name, client, ip_address, last_seen, max_bit_rate, transcoding_id from player;

drop table player;

alter table player_dg_tmp rename to player;
`)
	return err
}

func updatePlaylist_20200608153717(tx *sql.Tx) error {
	_, err := tx.Exec(`
create table playlist_dg_tmp
(
	id varchar(255) not null
		primary key,
	name varchar(255) default '' not null,
	comment varchar(255) default '' not null,
	duration real default 0 not null,
	song_count integer default 0 not null,
	owner varchar(255) default '' not null
		constraint playlist_user_user_name_fk
			references user (user_name)
				on update cascade on delete cascade,
	public bool default FALSE not null,
	created_at datetime,
	updated_at datetime
);

insert into playlist_dg_tmp(id, name, comment, duration, song_count, owner, public, created_at, updated_at) select id, name, comment, duration, song_count, owner, public, created_at, updated_at from playlist;

drop table playlist;

alter table playlist_dg_tmp rename to playlist;

create index playlist_name
	on playlist (name);
`)
	return err
}

func updatePlaylistTracks_20200608153717(tx *sql.Tx) error {
	_, err := tx.Exec(`
create table playlist_tracks_dg_tmp
(
	id integer default 0 not null,
	playlist_id varchar(255) not null
		constraint playlist_tracks_playlist_id_fk
			references playlist
				on update cascade on delete cascade,
	media_file_id varchar(255) not null
);

insert into playlist_tracks_dg_tmp(id, playlist_id, media_file_id) select id, playlist_id, media_file_id from playlist_tracks;

drop table playlist_tracks;

alter table playlist_tracks_dg_tmp rename to playlist_tracks;

create unique index playlist_tracks_pos
	on playlist_tracks (playlist_id, id);

`)
	return err
}

func Down20200608153717(_ context.Context, tx *sql.Tx) error {
	return nil
}

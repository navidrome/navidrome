package migrations

import (
	"context"
	"database/sql"

	"github.com/navidrome/navidrome/conf"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20200423204116, Down20200423204116)
}

func Up20200423204116(_ context.Context, tx *sql.Tx) error {

	if conf.Server.DbDriver == "pgx" {
		_, err := tx.Exec(`
		create collation nocase (
			provider = icu,
			locale = 'und-u-ks-level2',
			deterministic = false
			);
		`)
		if err != nil {
			return err
		}
	}

	_, err := tx.Exec(`
alter table artist
	add order_artist_name varchar(255) collate nocase;
alter table artist
	add sort_artist_name varchar(255) collate nocase;
create index if not exists artist_order_artist_name
	on artist (order_artist_name);

alter table album
	add order_album_name varchar(255) collate nocase;
alter table album
	add order_album_artist_name varchar(255) collate nocase;
alter table album
	add sort_album_name varchar(255) collate nocase;
alter table album
	add sort_artist_name varchar(255) collate nocase;
alter table album
	add sort_album_artist_name varchar(255) collate nocase;
create index if not exists album_order_album_name
	on album (order_album_name);
create index if not exists album_order_album_artist_name
	on album (order_album_artist_name);

alter table media_file
	add order_album_name varchar(255) collate nocase;
alter table media_file
	add order_album_artist_name varchar(255) collate nocase;
alter table media_file
	add order_artist_name varchar(255) collate nocase;
alter table media_file
	add sort_album_name varchar(255) collate nocase;
alter table media_file
	add sort_artist_name varchar(255) collate nocase;
alter table media_file
	add sort_album_artist_name varchar(255) collate nocase;
alter table media_file
	add sort_title varchar(255) collate nocase;
create index if not exists media_file_order_album_name
	on media_file (order_album_name);
create index if not exists media_file_order_artist_name
	on media_file (order_artist_name);
`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan will be performed to change the search behaviour")
	return forceFullRescan(tx)
}

func Down20200423204116(_ context.Context, tx *sql.Tx) error {

	if conf.Server.DbDriver == "pgx" {
		_, err := tx.Exec(`
		drop collation nocase;
		`)
		if err != nil {
			return err
		}
	}

	return nil
}

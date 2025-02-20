package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/navidrome/navidrome/conf"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddLibraryTable, downAddLibraryTable)
}

func upAddLibraryTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		create table library (
			id integer primary key autoincrement,
			name text not null unique,
			path text not null unique,
			remote_path text null default '',
			last_scan_at datetime not null default '0000-00-00 00:00:00',
			updated_at datetime not null default current_timestamp,
			created_at datetime not null default current_timestamp
		);`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		insert into library(id, name, path) values(1, 'Music Library', '%s');
		delete from property where id like 'LastScan-%%';
`, conf.Server.MusicFolder))
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		alter table media_file add column library_id integer not null default 1 
		   references library(id) on delete cascade;
		alter table album add column library_id integer not null default 1 
		   references library(id) on delete cascade;

		create table if not exists  library_artist
		(
			library_id integer not null default 1
				references library(id)
					on delete cascade,
			artist_id varchar not null default null
				references artist(id)
					on delete cascade,
			constraint library_artist_ux
				unique (library_id, artist_id)
		);

		insert into library_artist(library_id, artist_id) select 1, id from artist;
`)

	return err
}

func downAddLibraryTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		alter table media_file drop column library_id;
		alter table album drop column library_id;
		drop table library_artist;	
		drop table library;
`)
	return err
}

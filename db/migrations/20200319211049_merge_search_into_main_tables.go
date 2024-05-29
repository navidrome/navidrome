package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20200319211049, Down20200319211049)
}

func Up20200319211049(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file
	add full_text varchar(255) default '';
create index if not exists media_file_full_text
	on media_file (full_text);

alter table album
	add full_text varchar(255) default '';
create index if not exists album_full_text
	on album (full_text);

alter table artist
	add full_text varchar(255) default '';
create index if not exists artist_full_text
	on artist (full_text);

drop table if exists search;
`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan will be performed!")
	return forceFullRescan(tx)
}

func Down20200319211049(_ context.Context, tx *sql.Tx) error {
	return nil
}

package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20201021085410, Down20201021085410)
}

func Up20201021085410(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file
	add mbz_track_id varchar(255);
alter table media_file
	add mbz_album_id varchar(255);
alter table media_file
	add mbz_artist_id varchar(255);
alter table media_file
	add mbz_album_artist_id varchar(255);
alter table media_file
	add mbz_album_type varchar(255);
alter table media_file
	add mbz_album_comment varchar(255);
alter table media_file
	add catalog_num varchar(255);

alter table album
	add mbz_album_id varchar(255);
alter table album
	add mbz_album_artist_id varchar(255);
alter table album
	add mbz_album_type varchar(255);
alter table album
	add mbz_album_comment varchar(255);
alter table album
	add catalog_num varchar(255);

create index if not exists album_mbz_album_type
	on album (mbz_album_type);

alter table artist
	add mbz_artist_id varchar(255);

`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan needs to be performed to import more tags")
	return forceFullRescan(tx)
}

func Down20201021085410(_ context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}

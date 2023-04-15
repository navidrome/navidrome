package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upAddAlbumInfo, downAddAlbumInfo)
}

func upAddAlbumInfo(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table album
	add description varchar(255) default '' not null;
alter table album
	add small_image_url varchar(255) default '' not null;
alter table album
	add medium_image_url varchar(255) default '' not null;
alter table album
	add large_image_url varchar(255) default '' not null;
alter table album
	add external_url varchar(255) default '' not null;
alter table album
	add external_info_updated_at datetime;
`)
	return err
}

func downAddAlbumInfo(tx *sql.Tx) error {
	return nil
}

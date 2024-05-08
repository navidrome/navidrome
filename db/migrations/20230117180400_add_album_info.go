package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddAlbumInfo, downAddAlbumInfo)
}

func upAddAlbumInfo(_ context.Context, tx *sql.Tx) error {
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

func downAddAlbumInfo(_ context.Context, tx *sql.Tx) error {
	return nil
}

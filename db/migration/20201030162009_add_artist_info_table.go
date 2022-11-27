package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(upAddArtistImageUrl, downAddArtistImageUrl)
}

func upAddArtistImageUrl(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table artist
	add biography varchar(255) default '' not null;
alter table artist
	add small_image_url varchar(255) default '' not null;
alter table artist
	add medium_image_url varchar(255) default '' not null;
alter table artist
	add large_image_url varchar(255) default '' not null;
alter table artist
	add similar_artists varchar(255) default '' not null;
alter table artist
	add external_url varchar(255) default '' not null;
alter table artist
	add external_info_updated_at datetime;
`)
	return err
}

func downAddArtistImageUrl(tx *sql.Tx) error {
	return nil
}

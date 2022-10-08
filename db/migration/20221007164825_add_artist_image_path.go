package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(upAddArtistImagePath, downAddArtistImagePath)
}

func upAddArtistImagePath(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec(`
alter table album
	add artist_image_path varchar(255) default '' not null;

alter table artist
	add image_path varchar(255) default '' not null;

alter table artist
	add image_id	varchar(255) default '' not null;

alter table artist add updated_at datetime;
`)

	if err != nil {
		return err
	}
	notice(tx, "A full rescan needs to be performed to import more tags")
	return forceFullRescan(tx)
}

func downAddArtistImagePath(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}

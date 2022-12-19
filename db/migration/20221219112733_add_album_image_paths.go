package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(upAddAlbumImagePaths, downAddAlbumImagePaths)
}

func upAddAlbumImagePaths(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table main.album add image_files varchar;
`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan needs to be performed to import all album images")
	return forceFullRescan(tx)
}

func downAddAlbumImagePaths(tx *sql.Tx) error {
	return nil
}

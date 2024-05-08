package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddAlbumImagePaths, downAddAlbumImagePaths)
}

func upAddAlbumImagePaths(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table main.album add image_files varchar;
`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan needs to be performed to import all album images")
	return forceFullRescan(tx)
}

func downAddAlbumImagePaths(_ context.Context, tx *sql.Tx) error {
	return nil
}

package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddAlbumImagePaths, downAddAlbumImagePaths)
}

func upAddAlbumImagePaths(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
alter table main.album add image_files varchar;
`)
	if err != nil {
		return err
	}
	notice(ctx, tx, "A full rescan needs to be performed to import all album images")
	return forceFullRescan(ctx, tx)
}

func downAddAlbumImagePaths(_ context.Context, _ *sql.Tx) error {
	return nil
}

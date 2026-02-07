package migrations

import (
	"context"
	"database/sql"

	"github.com/navidrome/navidrome/db/dialect"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upRemoveDanglingItems, downRemoveDanglingItems)
}

func upRemoveDanglingItems(ctx context.Context, tx *sql.Tx) error {
	missingTrue := "1"
	if dialect.Current != nil && dialect.Current.Name() == "postgres" {
		missingTrue = "true"
	}

	_, err := tx.ExecContext(ctx, `
update media_file set missing = `+missingTrue+` where folder_id = '';
update album set missing = `+missingTrue+` where folder_ids = '[]';
`)
	return err
}

func downRemoveDanglingItems(_ context.Context, _ *sql.Tx) error {
	return nil
}

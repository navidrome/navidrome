package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upRemoveInvalidArtistIds, downRemoveInvalidArtistIds)
}

func upRemoveInvalidArtistIds(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
update media_file set artist_id = '' where not exists(select 1 from artist where id = artist_id)
`)
	return err
}

func downRemoveInvalidArtistIds(_ context.Context, _ *sql.Tx) error {
	return nil
}

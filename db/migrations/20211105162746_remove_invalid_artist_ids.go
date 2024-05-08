package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upRemoveInvalidArtistIds, downRemoveInvalidArtistIds)
}

func upRemoveInvalidArtistIds(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
update media_file set artist_id = '' where not exists(select 1 from artist where id = artist_id)
`)
	return err
}

func downRemoveInvalidArtistIds(_ context.Context, tx *sql.Tx) error {
	return nil
}

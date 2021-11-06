package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(upRemoveInvalidArtistIds, downRemoveInvalidArtistIds)
}

func upRemoveInvalidArtistIds(tx *sql.Tx) error {
	_, err := tx.Exec(`
update media_file set artist_id = '' where not exists(select 1 from artist where id = artist_id)
`)
	return err
}

func downRemoveInvalidArtistIds(tx *sql.Tx) error {
	return nil
}

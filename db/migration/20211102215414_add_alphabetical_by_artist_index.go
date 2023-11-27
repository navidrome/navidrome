package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddAlphabeticalByArtistIndex, downAddAlphabeticalByArtistIndex)
}

func upAddAlphabeticalByArtistIndex(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
create index album_alphabetical_by_artist 
    ON album(compilation, order_album_artist_name, order_album_name)
`)
	return err
}

func downAddAlphabeticalByArtistIndex(_ context.Context, tx *sql.Tx) error {
	return nil
}

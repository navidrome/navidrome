package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(upAddAlphabeticalByArtistIndex, downAddAlphabeticalByArtistIndex)
}

func upAddAlphabeticalByArtistIndex(tx *sql.Tx) error {
	_, err := tx.Exec(`
create index album_alphabetical_by_artist 
    ON album(compilation, order_album_artist_name, order_album_name)
`)
	return err
}

func downAddAlphabeticalByArtistIndex(tx *sql.Tx) error {
	return nil
}

package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upTouchPlaylists, downTouchPlaylists)
}

func upTouchPlaylists(tx *sql.Tx) error {
	_, err := tx.Exec(`update playlist set updated_at = datetime('now');`)
	return err
}

func downTouchPlaylists(tx *sql.Tx) error {
	return nil
}

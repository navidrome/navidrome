package migrations

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(upAddApikeyColumnToPlayers, downAddApikeyColumnToPlayers)
}

func upAddApikeyColumnToPlayers(tx *sql.Tx) error {
	_, err := tx.Exec(`alter table player
    add api_key VARCHAR(255);`)

	return err
}

func downAddApikeyColumnToPlayers(tx *sql.Tx) error {
	_, err := tx.Exec(`alter table player
    drop api_key VARCHAR(255);`)

	return err
}

package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upUpdateShareFieldNames, downUpdateShareFieldNames)
}

func upUpdateShareFieldNames(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table share rename column expires to expires_at;
alter table share rename column created to created_at;
alter table share rename column last_visited to last_visited_at;
`)

	return err
}

func downUpdateShareFieldNames(tx *sql.Tx) error {
	return nil
}

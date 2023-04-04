package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(Up20200508093059, Down20200508093059)
}

func Up20200508093059(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table artist
	add song_count integer default 0 not null;
`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan will be performed to calculate artists' song counts")
	return forceFullRescan(tx)
}

func Down20200508093059(tx *sql.Tx) error {
	return nil
}

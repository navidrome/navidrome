package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20200508093059, Down20200508093059)
}

func Up20200508093059(_ context.Context, tx *sql.Tx) error {
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

func Down20200508093059(_ context.Context, tx *sql.Tx) error {
	return nil
}

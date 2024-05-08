package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20201110205344, Down20201110205344)
}

func Up20201110205344(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file
	add comment varchar;
alter table media_file
	add lyrics varchar;

alter table album
	add comment varchar;
`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan will be performed to import comments and lyrics")
	return forceFullRescan(tx)
}

func Down20201110205344(_ context.Context, tx *sql.Tx) error {
	return nil
}

package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20201003111749, Down20201003111749)
}

func Up20201003111749(tx *sql.Tx) error {
	_, err := tx.Exec(`
create index if not exists annotation_starred_at
	on annotation (starred_at);
    `)
	return err
}

func Down20201003111749(tx *sql.Tx) error {
	return nil
}

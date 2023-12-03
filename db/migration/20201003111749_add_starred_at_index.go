package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20201003111749, Down20201003111749)
}

func Up20201003111749(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
create index if not exists annotation_starred_at
	on annotation (starred_at);
    `)
	return err
}

func Down20201003111749(_ context.Context, tx *sql.Tx) error {
	return nil
}

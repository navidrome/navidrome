package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20200411164603, Down20200411164603)
}

func Up20200411164603(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table playlist
	add created_at datetime;
alter table playlist
	add updated_at datetime;
update playlist 
	set created_at = datetime('now'), updated_at = datetime('now');
`)
	return err
}

func Down20200411164603(_ context.Context, tx *sql.Tx) error {
	return nil
}

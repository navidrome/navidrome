package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(Up20200411164603, Down20200411164603)
}

func Up20200411164603(tx *sql.Tx) error {
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

func Down20200411164603(tx *sql.Tx) error {
	return nil
}

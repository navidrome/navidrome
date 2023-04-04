package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(Up20201021135455, Down20201021135455)
}

func Up20201021135455(tx *sql.Tx) error {
	_, err := tx.Exec(`
create index if not exists media_file_artist_id
	on media_file (artist_id);
`)
	return err
}

func Down20201021135455(tx *sql.Tx) error {
	return nil
}

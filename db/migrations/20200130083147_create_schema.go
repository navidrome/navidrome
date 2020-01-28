package migrations

import (
	"database/sql"

	"github.com/deluan/navidrome/log"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200130083147, Down20200130083147)
}

func Up20200130083147(tx *sql.Tx) error {
	log.Info("Creating DB Schema")
	_, err := tx.Exec(schema)
	return err
}

func Down20200130083147(tx *sql.Tx) error {
	return nil
}

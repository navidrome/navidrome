package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(upAddDownloadToShare, downAddDownloadToShare)
}

func upAddDownloadToShare(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table share
	add downloadable bool not null default false;
`)
	return err
}

func downAddDownloadToShare(tx *sql.Tx) error {
	return nil
}

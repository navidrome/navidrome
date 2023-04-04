package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upRemoveCoverArtId, downRemoveCoverArtId)
}

func upRemoveCoverArtId(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table album drop column cover_art_id;
alter table album rename column cover_art_path to embed_art_path
`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan needs to be performed to import all album images")
	return forceFullRescan(tx)
}

func downRemoveCoverArtId(tx *sql.Tx) error {
	return nil
}

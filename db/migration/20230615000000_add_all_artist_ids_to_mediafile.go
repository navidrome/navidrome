package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(Up20231116000000, Down20231116000000)
}

func Up20231116000000(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file
	add all_artist_ids varchar;

create index if not exists mediafile_all_artist_ids
	on media_file (all_artist_ids);

`)
	if err != nil {
		return err
	}

	notice(tx, "A full rescan needs to be performed to import all artist IDs")
	return forceFullRescan(tx)
}

func Down20231116000000(tx *sql.Tx) error {
	return nil
}

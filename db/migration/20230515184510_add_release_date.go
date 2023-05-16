package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upAddRelRecYear, downAddRelRecYear)
}

func upAddRelRecYear(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file
    add date varchar(255) default '' not null;
alter table media_file
    add original_year int default 0 not null;
alter table media_file
    add original_date varchar(255) default '' not null;
alter table media_file
    add release_year int default 0 not null;
alter table media_file
    add release_date varchar(255) default '' not null;

alter table album
    add date varchar(255) default '' not null;
alter table album
    add min_original_year int default 0 not null;
alter table album
    add max_original_year int default 0 not null;
alter table album
    add original_date varchar(255) default '' not null;
alter table album
    add release_date varchar(255) default '' not null;
alter table album
    add releases integer default 0 not null;
`)
	if err != nil {
		return err
	}

	notice(tx, "A full rescan needs to be performed to import more tags")
	return forceFullRescan(tx)
}

func downAddRelRecYear(tx *sql.Tx) error {
	return nil
}

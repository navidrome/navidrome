package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(upAddRadioLinks, downAddRadioLinks)
}

func upAddRadioLinks(tx *sql.Tx) error {
	_, err := tx.Exec(`
create table if not exists radio_link
(
	id       varchar(255) primary key,
	name     varchar not null,
	url      varchar not null,
	radio_id varchar(255) not null 
		references radio (id)
			on update cascade on delete cascade
);

alter table radio add column is_playlist bool default false not null;
`)

	return err
}

func downAddRadioLinks(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}

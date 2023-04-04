package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upCreateSharesTable, downCreateSharesTable)
}

func upCreateSharesTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
create table share
(
	id             varchar(255) not null primary key,
	name           varchar(255) not null unique,
	description    varchar(255),
	expires        datetime,
	created        datetime,
	last_visited   datetime,
	resource_ids   varchar not null,
	resource_type  varchar(255) not null,
	visit_count    integer default 0
);
`)

	return err
}

func downCreateSharesTable(tx *sql.Tx) error {
	return nil
}

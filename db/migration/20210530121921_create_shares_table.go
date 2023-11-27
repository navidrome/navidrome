package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upCreateSharesTable, downCreateSharesTable)
}

func upCreateSharesTable(_ context.Context, tx *sql.Tx) error {
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

func downCreateSharesTable(_ context.Context, tx *sql.Tx) error {
	return nil
}

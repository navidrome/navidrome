package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upCreatePlayQueuesTable, downCreatePlayQueuesTable)
}

func upCreatePlayQueuesTable(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
create table playqueue
(
    id         varchar(255) not null primary key,
    user_id    varchar(255) not null
            references user (id)
            on update cascade on delete cascade,
	comment    varchar(255),
    current    varchar(255) not null,
    position   integer,
    changed_by varchar(255),
    items      varchar(255),
    created_at datetime,
    updated_at datetime
);
`)

	return err
}

func downCreatePlayQueuesTable(_ context.Context, tx *sql.Tx) error {
	return nil
}

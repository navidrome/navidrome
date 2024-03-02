package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddMissingShareInfo, downAddMissingShareInfo)
}

func upAddMissingShareInfo(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
drop table if exists share;
create table share
(
    id              varchar(255) not null
        primary key,
    description     varchar(255),
    expires_at      timestamp,
    last_visited_at timestamp,
    resource_ids    varchar      not null,
    resource_type   varchar(255) not null,
    contents        varchar,
    format 			varchar,
	max_bit_rate 	integer,
    visit_count     integer default 0,
    created_at      timestamp,
    updated_at      timestamp,
    user_id         varchar(255) not null
        constraint share_user_id_fk
            references "user"
);
`)
	return err
}

func downAddMissingShareInfo(_ context.Context, tx *sql.Tx) error {
	return nil
}

package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upCreateBookmarkTable, downCreateBookmarkTable)
}

func upCreateBookmarkTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
create table bookmark
(
    user_id    varchar(255) not null
        references user
            on update cascade on delete cascade,
    item_id    varchar(255) not null,
    item_type  varchar(255) not null,
    comment    varchar(255),
    position   integer,
    changed_by varchar(255),
    created_at datetime,
    updated_at datetime,
    constraint bookmark_pk
        unique (user_id, item_id, item_type)
);

create table playqueue_dg_tmp
(
	id varchar(255) not null,
	user_id varchar(255) not null
		references user
			on update cascade on delete cascade,
	current varchar(255),
	position real,
	changed_by varchar(255),
	items varchar(255),
	created_at datetime,
	updated_at datetime
);
drop table playqueue;
alter table playqueue_dg_tmp rename to playqueue;
`)

	return err
}

func downCreateBookmarkTable(tx *sql.Tx) error {
	return nil
}

package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddReferentialIntegrityToUserProps, downAddReferentialIntegrityToUserProps)
}

func upAddReferentialIntegrityToUserProps(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
create table user_props_dg_tmp
(
	user_id varchar not null
		constraint user_props_user_id_fk
			references user
				on update cascade on delete cascade,
	key varchar not null,
	value varchar,
	constraint user_props_pk
		primary key (user_id, key)
);

insert into user_props_dg_tmp(user_id, key, value) select user_id, key, value from user_props;

drop table user_props;

alter table user_props_dg_tmp rename to user_props;
`)
	return err
}

func downAddReferentialIntegrityToUserProps(_ context.Context, tx *sql.Tx) error {
	return nil
}

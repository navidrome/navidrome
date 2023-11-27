package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upDropEmailUniqueConstraint, downDropEmailUniqueConstraint)
}

func upDropEmailUniqueConstraint(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
create table user_dg_tmp
(
	id varchar(255) not null
		primary key,
	user_name varchar(255) default '' not null
		unique,
	name varchar(255) default '' not null,
	email varchar(255) default '' not null,
	password varchar(255) default '' not null,
	is_admin bool default FALSE not null,
	last_login_at datetime,
	last_access_at datetime,
	created_at datetime not null,
	updated_at datetime not null
);

insert into user_dg_tmp(id, user_name, name, email, password, is_admin, last_login_at, last_access_at, created_at, updated_at) select id, user_name, name, email, password, is_admin, last_login_at, last_access_at, created_at, updated_at from user;

drop table user;

alter table user_dg_tmp rename to user;
`)
	return err
}

func downDropEmailUniqueConstraint(_ context.Context, tx *sql.Tx) error {
	return nil
}

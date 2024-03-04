package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddSyncPlayqueueColumnToUserTable, downAddSyncPlayqueueColumnToUserTable)
}

func upAddSyncPlayqueueColumnToUserTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
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
		updated_at datetime not null,
		sync_playqueue bool default FALSE not null
	);

	insert into user_dg_tmp(id, user_name, name, email, password, is_admin, last_login_at, last_access_at, created_at, updated_at) select id, user_name, name, email, password, is_admin, last_login_at, last_access_at, created_at, updated_at from user;

	drop table user;

	alter table user_dg_tmp rename to user;
	`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan needs to be performed to import more tags")
	return forceFullRescan(tx)
}

func downAddSyncPlayqueueColumnToUserTable(ctx context.Context, tx *sql.Tx) error {
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
		updated_at datetime not null,
	);

	insert into user_dg_tmp(id, user_name, name, email, password, is_admin, last_login_at, last_access_at, created_at, updated_at) select id, user_name, name, email, password, is_admin, last_login_at, last_access_at, created_at, updated_at from user;

	drop table user;

	alter table user_dg_tmp rename to user;
	`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan needs to be performed to import more tags")
	return forceFullRescan(tx)
}

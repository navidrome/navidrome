package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upDropPlayerNameUniqueConstraint, downDropPlayerNameUniqueConstraint)
}

func upDropPlayerNameUniqueConstraint(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
create table player_dg_tmp
(
	id varchar(255) not null
		primary key,
	name varchar not null,
	user_agent varchar,
	user_name varchar not null
		references user (user_name)
			on update cascade on delete cascade,
	client varchar not null,
	ip_address varchar,
	last_seen timestamp,
	max_bit_rate int default 0,
	transcoding_id varchar,
	report_real_path bool default FALSE not null
);

insert into player_dg_tmp(id, name, user_agent, user_name, client, ip_address, last_seen, max_bit_rate, transcoding_id, report_real_path) select id, name, type, user_name, client, ip_address, last_seen, max_bit_rate, transcoding_id, report_real_path from player;

drop table player;

alter table player_dg_tmp rename to player;
create index if not exists player_match
	on player (client, user_agent, user_name);
create index if not exists player_name
	on player (name);
`)
	return err
}

func downDropPlayerNameUniqueConstraint(_ context.Context, tx *sql.Tx) error {
	return nil
}

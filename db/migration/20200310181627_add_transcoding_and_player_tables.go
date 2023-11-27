package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20200310181627, Down20200310181627)
}

func Up20200310181627(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
create table transcoding
(
	id varchar(255) not null primary key,
	name varchar(255) not null,
	target_format varchar(255) not null,
	command varchar(255) default '' not null,
	default_bit_rate int default 192,
	unique (name),
	unique (target_format)
);

create table player 
(
	id varchar(255) not null primary key,
    name varchar not null,
    type varchar,
    user_name varchar not null, 
    client varchar not null, 
    ip_address varchar,
    last_seen timestamp,
    max_bit_rate int default 0,
    transcoding_id varchar,
	unique (name),
	foreign key (transcoding_id)
	   references transcoding(id)
		  on update restrict 
		  on delete restrict
);
`)
	return err
}

func Down20200310181627(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
drop table transcoding;
drop table player;
`)
	return err
}

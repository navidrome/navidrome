package migration

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200310181627, Down20200310181627)
}

func Up20200310181627(tx *sql.Tx) error {
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
    transcoding_id varchar, -- todo foreign key 
    max_bit_rate int default 0,
	unique (name)
);
`)
	return err
}

func Down20200310181627(tx *sql.Tx) error {
	_, err := tx.Exec(`
drop table transcoding;
drop table player;
`)
	return err
}

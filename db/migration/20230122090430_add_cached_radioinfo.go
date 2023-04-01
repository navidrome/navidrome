package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(upAddCachedRadioinfo, downAddCachedRadioinfo)
}

func upAddCachedRadioinfo(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE radio RENAME to radio_tmp;

CREATE TABLE IF NOT EXISTS radioinfo (
	id            varchar(255) not null primary key,
	name          varchar not null,
	url           varchar not null,
	homepage      varchar not null,
	favicon       varchar not null,
	tags          varchar not null,
	country       varchar not null,
	country_code  varchar not null,
	codec         varchar not null,
	bitrate       integer not null
);

CREATE TABLE IF NOT EXISTS radio
(
  id            varchar(255) not null primary key,
	name          varchar not null unique,
	stream_url    varchar not null,
	home_page_url varchar default '' not null,
	favicon       varchar default '' not null,
	tags          varchar default '' not null,
	country       varchar default '' not null,
	country_code  varchar default '' not null,
	codec         varchar default '' not null,
	bitrate       integer default 0  not null,
	radioinfo_id  varchar(255)
		references radioinfo (id)
			on update cascade on delete set null,
	created_at    datetime,
	updated_at    datetime
);

INSERT INTO radio (id, name, stream_url, home_page_url, created_at, updated_at) SELECT 
	id, name, stream_url, home_page_url, created_at, updated_at 
FROM radio_tmp;

DROP TABLE radio_tmp;
	`)
	return err
}

func downAddCachedRadioinfo(tx *sql.Tx) error {
	return nil
}

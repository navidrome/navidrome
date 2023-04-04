package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upCreateInternetRadio, downCreateInternetRadio)
}

func upCreateInternetRadio(tx *sql.Tx) error {
	_, err := tx.Exec(`
create table if not exists radio
(
    id            varchar(255) not null primary key,
	name          varchar not null unique,
	stream_url    varchar not null,
	home_page_url varchar default '' not null,
	created_at    datetime,
	updated_at    datetime
);
`)
	return err
}

func downCreateInternetRadio(tx *sql.Tx) error {
	return nil
}

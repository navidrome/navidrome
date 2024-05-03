package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddMediafileIsrc, downAddMediafileIsrc)
}

func upAddMediafileIsrc(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
alter table media_file
	add isrc varchar default '';
alter table media_file
	add upc varchar default '';
alter table album
	add upc varchar default '';
	`)
	return err
}

func downAddMediafileIsrc(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
alter table media_file
	drop isrc;
alter table media_file
	drop upc;
alter table album
	drop upc;
	`)
	return err
}

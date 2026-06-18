package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upUpdateShareFieldNames, downUpdateShareFieldNames)
}

func upUpdateShareFieldNames(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
alter table share rename column expires to expires_at;
alter table share rename column created to created_at;
alter table share rename column last_visited to last_visited_at;
`)

	return err
}

func downUpdateShareFieldNames(_ context.Context, _ *sql.Tx) error {
	return nil
}

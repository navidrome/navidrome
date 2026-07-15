package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20200508093059, Down20200508093059)
}

func Up20200508093059(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
alter table artist
	add song_count integer default 0 not null;
`)
	if err != nil {
		return err
	}
	notice(ctx, tx, "A full rescan will be performed to calculate artists' song counts")
	return forceFullRescan(ctx, tx)
}

func Down20200508093059(_ context.Context, _ *sql.Tx) error {
	return nil
}

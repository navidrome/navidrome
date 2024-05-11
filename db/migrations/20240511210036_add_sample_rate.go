package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddSampleRate, downAddSampleRate)
}

func upAddSampleRate(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
alter table media_file
    add sample_rate integer not null default 0;

create index if not exists media_file_sample_rate
	on media_file (sample_rate);
`)
	notice(tx, "A full rescan should be performed to pick up additional tags")
	return err
}

func downAddSampleRate(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `alter table media_file drop sample_rate;`)
	return err
}

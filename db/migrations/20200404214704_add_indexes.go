package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20200404214704, Down20200404214704)
}

func Up20200404214704(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
create index if not exists media_file_year
	on media_file (year);

create index if not exists media_file_duration
	on media_file (duration);

create index if not exists media_file_track_number
	on media_file (disc_number, track_number);
`)
	return err
}

func Down20200404214704(_ context.Context, tx *sql.Tx) error {
	return nil
}

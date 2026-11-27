package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20200512104202, Down20200512104202)
}

func Up20200512104202(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
alter table media_file 
    add disc_subtitle varchar(255);
    `)
	if err != nil {
		return err
	}
	notice(ctx, tx, "A full rescan will be performed to import disc subtitles")
	return forceFullRescan(ctx, tx)
}

func Down20200512104202(_ context.Context, _ *sql.Tx) error {
	return nil
}

package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upCreateScrobblesTable, downCreateScrobblesTable)
}

func upCreateScrobblesTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
CREATE TABLE scrobbles(
    media_file_id VARCHAR(255) NOT NULL
        REFERENCES media_file(id)
            ON DELETE CASCADE
            ON UPDATE CASCADE,
    user_id VARCHAR(255) NOT NULL
        REFERENCES "user"(id)
            ON DELETE CASCADE
            ON UPDATE CASCADE,
    submission_time INTEGER NOT NULL
);
CREATE INDEX scrobbles_date ON scrobbles (submission_time);
`)
	return err
}

func downCreateScrobblesTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP TABLE scrobbles;`)
	return err
}

package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upCreatePluginTable, downCreatePluginTable)
}

func upCreatePluginTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, adaptSQL(`
CREATE TABLE IF NOT EXISTS plugin (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    manifest JSONB NOT NULL,
    config JSONB,
    users JSONB,
    all_users BOOL NOT NULL DEFAULT false,
    libraries JSONB,
    all_libraries BOOL NOT NULL DEFAULT false,
    enabled BOOL NOT NULL DEFAULT false,
    last_error TEXT,
    sha256 TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
`))
	return err
}

func downCreatePluginTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS plugin;`)
	return err
}

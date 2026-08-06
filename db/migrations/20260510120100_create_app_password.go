package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upCreateAppPassword, downCreateAppPassword)
}

func upCreateAppPassword(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS app_password (
    id               VARCHAR(255) NOT NULL PRIMARY KEY,
    user_id          VARCHAR(255) NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    name             VARCHAR(255) NOT NULL,
    secret_encrypted TEXT NOT NULL,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at     DATETIME,
    expires_at       DATETIME
);
CREATE INDEX IF NOT EXISTS app_password_user_id ON app_password(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS app_password_user_name ON app_password(user_id, name);`)
	return err
}

func downCreateAppPassword(ctx context.Context, tx *sql.Tx) error {
	return nil
}

package migrations

import (
	"context"
	"database/sql"

	"github.com/navidrome/navidrome/conf"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upTouchPlaylists, downTouchPlaylists)
}

func upTouchPlaylists(_ context.Context, tx *sql.Tx) error {
	var err error
	switch conf.Server.DbDriver {
	case "sqlite3":
		_, err = tx.Exec(`update playlist set updated_at = datetime('now');`)
	case "pgx":
		_, err = tx.Exec(`update playlist set updated_at = now();`)
	}

	return err
}

func downTouchPlaylists(_ context.Context, tx *sql.Tx) error {
	return nil
}

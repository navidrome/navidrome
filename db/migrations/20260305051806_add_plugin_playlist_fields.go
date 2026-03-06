package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddPluginPlaylistFields, downAddPluginPlaylistFields)
}

func upAddPluginPlaylistFields(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
ALTER TABLE playlist ADD COLUMN plugin_id VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE playlist ADD COLUMN plugin_playlist_id VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE playlist ADD COLUMN valid_until DATETIME DEFAULT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_playlist_plugin ON playlist(plugin_id, plugin_playlist_id, owner_id) WHERE plugin_id != '';
`)
	return err
}

func downAddPluginPlaylistFields(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
DROP INDEX IF EXISTS idx_playlist_plugin;
ALTER TABLE playlist DROP COLUMN valid_until;
ALTER TABLE playlist DROP COLUMN plugin_playlist_id;
ALTER TABLE playlist DROP COLUMN plugin_id;
`)
	return err
}

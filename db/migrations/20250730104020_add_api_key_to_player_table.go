package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddAPIKeyToPlayer, downDropAPIKeyFromPlayer)
}

func upAddAPIKeyToPlayer(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
-- Add nullable api_key column to player table
ALTER TABLE player ADD COLUMN api_key VARCHAR(255);

-- Add index on api_key for faster lookups
CREATE INDEX IF NOT EXISTS player_api_key ON player(api_key);
`)
	return err
}

func downDropAPIKeyFromPlayer(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
-- Drop the index first
DROP INDEX IF EXISTS player_api_key;

-- Then drop the column
ALTER TABLE player DROP COLUMN api_key;
`)
	return err
}

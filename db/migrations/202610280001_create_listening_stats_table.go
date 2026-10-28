package migrations

import (
	"database/sql"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(0, "202610280001_create_listening_stats_table", func(tx *sql.DB, psql bool) error {
		_, err := tx.Exec(`
			CREATE TABLE IF NOT EXISTS listening_stats (
				id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				user_id       UUID NOT NULL,
				entity_type   VARCHAR(32) NOT NULL CHECK (entity_type IN ('artist', 'album', 'song')),
				entity_id     UUID NOT NULL,
				play_count    INTEGER NOT NULL DEFAULT 0,
				total_ms_played BIGINT NOT NULL DEFAULT 0,
				first_played  TIMESTAMP NOT NULL DEFAULT now(),
				last_played   TIMESTAMP NOT NULL DEFAULT now(),
				updated_at    TIMESTAMP NOT NULL DEFAULT now(),
				UNIQUE(user_id, entity_type, entity_id)
			);
			CREATE INDEX IF NOT EXISTS idx_listening_stats_user ON listening_stats(user_id, entity_type, total_ms_played DESC);
		`)
		return err
	})
}

package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddPostgresPerformanceIndexes, downAddPostgresPerformanceIndexes)
}

func upAddPostgresPerformanceIndexes(_ context.Context, tx *sql.Tx) error {
	if !IsPostgres() {
		return nil
	}

	// Enable pg_trgm extension for trigram-based LIKE '%...%' acceleration
	if _, err := tx.Exec(`CREATE EXTENSION IF NOT EXISTS pg_trgm`); err != nil {
		return err
	}

	// GIN trigram indexes for full_text LIKE searches
	for _, stmt := range []string{
		`CREATE INDEX IF NOT EXISTS idx_media_file_full_text_trgm ON media_file USING GIN (full_text gin_trgm_ops)`,
		`CREATE INDEX IF NOT EXISTS idx_album_full_text_trgm ON album USING GIN (full_text gin_trgm_ops)`,
		`CREATE INDEX IF NOT EXISTS idx_artist_full_text_trgm ON artist USING GIN (full_text gin_trgm_ops)`,

		// GIN indexes for JSONB column searches
		`CREATE INDEX IF NOT EXISTS idx_album_tags_gin ON album USING GIN (tags)`,
		`CREATE INDEX IF NOT EXISTS idx_media_file_tags_gin ON media_file USING GIN (tags)`,
		`CREATE INDEX IF NOT EXISTS idx_album_participants_gin ON album USING GIN (participants)`,
		`CREATE INDEX IF NOT EXISTS idx_media_file_participants_gin ON media_file USING GIN (participants)`,

		// Composite indexes for common filter patterns (library + missing)
		`CREATE INDEX IF NOT EXISTS idx_media_file_library_missing ON media_file (library_id, missing)`,
		`CREATE INDEX IF NOT EXISTS idx_album_library_missing ON album (library_id, missing)`,
		`CREATE INDEX IF NOT EXISTS idx_folder_library_missing ON folder (library_id, missing)`,
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

func downAddPostgresPerformanceIndexes(_ context.Context, tx *sql.Tx) error {
	if !IsPostgres() {
		return nil
	}

	for _, stmt := range []string{
		`DROP INDEX IF EXISTS idx_media_file_full_text_trgm`,
		`DROP INDEX IF EXISTS idx_album_full_text_trgm`,
		`DROP INDEX IF EXISTS idx_artist_full_text_trgm`,
		`DROP INDEX IF EXISTS idx_album_tags_gin`,
		`DROP INDEX IF EXISTS idx_media_file_tags_gin`,
		`DROP INDEX IF EXISTS idx_album_participants_gin`,
		`DROP INDEX IF EXISTS idx_media_file_participants_gin`,
		`DROP INDEX IF EXISTS idx_media_file_library_missing`,
		`DROP INDEX IF EXISTS idx_album_library_missing`,
		`DROP INDEX IF EXISTS idx_folder_library_missing`,
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

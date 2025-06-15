package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddUserLibraryTable, downAddUserLibraryTable)
}

func upAddUserLibraryTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
	-- Create user_library association table
		CREATE TABLE user_library (
			user_id VARCHAR(255) NOT NULL,
			library_id INTEGER NOT NULL,
			PRIMARY KEY (user_id, library_id),
			FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
			FOREIGN KEY (library_id) REFERENCES library(id) ON DELETE CASCADE
		);
	-- Create indexes for performance
		CREATE INDEX idx_user_library_user_id ON user_library(user_id);
		CREATE INDEX idx_user_library_library_id ON user_library(library_id);

	-- Populate with existing users having access to library ID 1 (existing setup)
	-- Admin users get access to all libraries, regular users get access to library 1
		INSERT INTO user_library (user_id, library_id)
		SELECT u.id, 1
		FROM user u;

	-- Add total_duration column to library table
		ALTER TABLE library ADD COLUMN total_duration INTEGER DEFAULT 0;
		UPDATE library SET total_duration = (
			SELECT IFNULL(SUM(duration),0) from album where album.library_id = library.id and missing = 0
		);
	`)

	return err
}

func downAddUserLibraryTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		DROP INDEX IF EXISTS idx_user_library_library_id;
		DROP INDEX IF EXISTS idx_user_library_user_id;
		DROP TABLE IF EXISTS user_library;
		ALTER TABLE library DROP COLUMN IF EXISTS total_duration;
	`)
	return err
}

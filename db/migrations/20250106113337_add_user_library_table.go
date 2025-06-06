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
	// Create user_library association table
	_, err := tx.ExecContext(ctx, `
		CREATE TABLE user_library (
			user_id VARCHAR(255) NOT NULL,
			library_id INTEGER NOT NULL,
			PRIMARY KEY (user_id, library_id),
			FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
			FOREIGN KEY (library_id) REFERENCES library(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	// Create indexes for performance
	_, err = tx.ExecContext(ctx, `
		CREATE INDEX idx_user_library_user_id ON user_library(user_id);
		CREATE INDEX idx_user_library_library_id ON user_library(library_id);
	`)
	if err != nil {
		return err
	}

	// Populate with existing users having access to library ID 1 (existing setup)
	// Admin users get access to all libraries, regular users get access to library 1
	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_library (user_id, library_id)
		SELECT u.id, 1
		FROM user u;
	`)

	return err
}

func downAddUserLibraryTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		DROP INDEX IF EXISTS idx_user_library_library_id;
		DROP INDEX IF EXISTS idx_user_library_user_id;
		DROP TABLE IF EXISTS user_library;
	`)
	return err
}

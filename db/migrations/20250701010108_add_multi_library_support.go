package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddMultiLibrarySupport, downAddMultiLibrarySupport)
}

func upAddMultiLibrarySupport(ctx context.Context, tx *sql.Tx) error {
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
		ALTER TABLE library ADD COLUMN total_duration real DEFAULT 0;
		UPDATE library SET total_duration = (
			SELECT IFNULL(SUM(duration),0) from album where album.library_id = library.id and missing = 0
		);

	-- Add default_new_users column to library table
		ALTER TABLE library ADD COLUMN default_new_users boolean DEFAULT false;
		-- Set library ID 1 (default library) as default for new users
		UPDATE library SET default_new_users = true WHERE id = 1;

	-- Add stats column to library_artist junction table for per-library artist statistics
		ALTER TABLE library_artist ADD COLUMN stats text DEFAULT '{}';
		
	-- Migrate existing global artist stats to per-library format in library_artist table
	-- For each library_artist association, copy the artist's global stats
		UPDATE library_artist 
		SET stats = (
			SELECT COALESCE(artist.stats, '{}')
			FROM artist 
			WHERE artist.id = library_artist.artist_id
		);

	-- Remove stats column from artist table to eliminate duplication
	-- Stats are now stored per-library in library_artist table
		ALTER TABLE artist DROP COLUMN stats;

	-- Create library_tag table for per-library tag statistics
		CREATE TABLE library_tag (
			tag_id VARCHAR NOT NULL,
			library_id INTEGER NOT NULL,
			album_count INTEGER DEFAULT 0 NOT NULL,
			media_file_count INTEGER DEFAULT 0 NOT NULL,
			PRIMARY KEY (tag_id, library_id),
			FOREIGN KEY (tag_id) REFERENCES tag(id) ON DELETE CASCADE,
			FOREIGN KEY (library_id) REFERENCES library(id) ON DELETE CASCADE
		);

	-- Create indexes for optimal query performance
		CREATE INDEX idx_library_tag_tag_id ON library_tag(tag_id);
		CREATE INDEX idx_library_tag_library_id ON library_tag(library_id);

	-- Migrate existing tag stats to per-library format in library_tag table
	-- For existing installations, copy current global stats to library ID 1 (default library)
		INSERT INTO library_tag (tag_id, library_id, album_count, media_file_count)
		SELECT t.id, 1, t.album_count, t.media_file_count
		FROM tag t
		WHERE EXISTS (SELECT 1 FROM library WHERE id = 1);

	-- Remove global stats from tag table as they are now per-library
		ALTER TABLE tag DROP COLUMN album_count;
		ALTER TABLE tag DROP COLUMN media_file_count;
	`)

	return err
}

func downAddMultiLibrarySupport(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		-- Restore stats column to artist table before removing from library_artist
		ALTER TABLE artist ADD COLUMN stats text DEFAULT '{}';
		
		-- Restore global stats by aggregating from library_artist (simplified approach)
		-- In a real rollback scenario, this might need more sophisticated logic
		UPDATE artist 
		SET stats = (
			SELECT COALESCE(la.stats, '{}')
			FROM library_artist la 
			WHERE la.artist_id = artist.id 
			LIMIT 1
		);
		
		ALTER TABLE library_artist DROP COLUMN IF EXISTS stats;
		DROP INDEX IF EXISTS idx_user_library_library_id;
		DROP INDEX IF EXISTS idx_user_library_user_id;
		DROP TABLE IF EXISTS user_library;
		ALTER TABLE library DROP COLUMN IF EXISTS total_duration;
		ALTER TABLE library DROP COLUMN IF EXISTS default_new_users;
		
		-- Drop library_tag table and its indexes
		DROP INDEX IF EXISTS idx_library_tag_library_id;
		DROP INDEX IF EXISTS idx_library_tag_tag_id;
		DROP TABLE IF EXISTS library_tag;
	`)
	return err
}

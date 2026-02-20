package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddFts5Search, downAddFts5Search)
}

func upAddFts5Search(ctx context.Context, tx *sql.Tx) error {
	notice(tx, "Adding FTS5 full-text search indexes. This may take a moment on large libraries.")

	// Step 1: Add search_participants column to media_file and album
	_, err := tx.ExecContext(ctx, `ALTER TABLE media_file ADD COLUMN search_participants TEXT NOT NULL DEFAULT ''`)
	if err != nil {
		return fmt.Errorf("adding search_participants to media_file: %w", err)
	}
	_, err = tx.ExecContext(ctx, `ALTER TABLE album ADD COLUMN search_participants TEXT NOT NULL DEFAULT ''`)
	if err != nil {
		return fmt.Errorf("adding search_participants to album: %w", err)
	}

	// Step 2: Populate search_participants from participants JSON.
	// Extract all "name" values from the participants JSON structure.
	// participants is a JSON object like: {"artist":[{"name":"...","id":"..."}],"albumartist":[...]}
	// We use json_each + json_extract to flatten all names into a space-separated string.
	_, err = tx.ExecContext(ctx, `
		UPDATE media_file SET search_participants = COALESCE(
			(SELECT group_concat(json_extract(je2.value, '$.name'), ' ')
			 FROM json_each(media_file.participants) AS je1,
			      json_each(je1.value) AS je2
			 WHERE json_extract(je2.value, '$.name') IS NOT NULL),
			''
		)
		WHERE participants IS NOT NULL AND participants != '' AND participants != '{}'
	`)
	if err != nil {
		return fmt.Errorf("populating media_file search_participants: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE album SET search_participants = COALESCE(
			(SELECT group_concat(json_extract(je2.value, '$.name'), ' ')
			 FROM json_each(album.participants) AS je1,
			      json_each(je1.value) AS je2
			 WHERE json_extract(je2.value, '$.name') IS NOT NULL),
			''
		)
		WHERE participants IS NOT NULL AND participants != '' AND participants != '{}'
	`)
	if err != nil {
		return fmt.Errorf("populating album search_participants: %w", err)
	}

	// Step 3: Create FTS5 virtual tables
	_, err = tx.ExecContext(ctx, `
		CREATE VIRTUAL TABLE IF NOT EXISTS media_file_fts USING fts5(
			title, album, artist, album_artist,
			sort_title, sort_album_name, sort_artist_name, sort_album_artist_name,
			disc_subtitle, search_participants,
			content='', content_rowid='rowid',
			tokenize='unicode61 remove_diacritics 2'
		)
	`)
	if err != nil {
		return fmt.Errorf("creating media_file_fts: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE VIRTUAL TABLE IF NOT EXISTS album_fts USING fts5(
			name, sort_album_name, album_artist,
			search_participants, discs, catalog_num, album_version,
			content='', content_rowid='rowid',
			tokenize='unicode61 remove_diacritics 2'
		)
	`)
	if err != nil {
		return fmt.Errorf("creating album_fts: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE VIRTUAL TABLE IF NOT EXISTS artist_fts USING fts5(
			name, sort_artist_name,
			content='', content_rowid='rowid',
			tokenize='unicode61 remove_diacritics 2'
		)
	`)
	if err != nil {
		return fmt.Errorf("creating artist_fts: %w", err)
	}

	// Step 4: Bulk-populate FTS5 indexes from existing data
	_, err = tx.ExecContext(ctx, `
		INSERT INTO media_file_fts(rowid, title, album, artist, album_artist,
			sort_title, sort_album_name, sort_artist_name, sort_album_artist_name,
			disc_subtitle, search_participants)
		SELECT rowid, title, album, artist, album_artist,
			sort_title, sort_album_name, sort_artist_name, sort_album_artist_name,
			COALESCE(disc_subtitle, ''), COALESCE(search_participants, '')
		FROM media_file
	`)
	if err != nil {
		return fmt.Errorf("populating media_file_fts: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO album_fts(rowid, name, sort_album_name, album_artist,
			search_participants, discs, catalog_num, album_version)
		SELECT rowid, name, COALESCE(sort_album_name, ''), COALESCE(album_artist, ''),
			COALESCE(search_participants, ''), COALESCE(discs, ''),
			COALESCE(catalog_num, ''), COALESCE(album_version, '')
		FROM album
	`)
	if err != nil {
		return fmt.Errorf("populating album_fts: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO artist_fts(rowid, name, sort_artist_name)
		SELECT rowid, name, COALESCE(sort_artist_name, '')
		FROM artist
	`)
	if err != nil {
		return fmt.Errorf("populating artist_fts: %w", err)
	}

	// Step 5: Create triggers for media_file
	_, err = tx.ExecContext(ctx, `
		CREATE TRIGGER media_file_fts_ai AFTER INSERT ON media_file BEGIN
			INSERT INTO media_file_fts(rowid, title, album, artist, album_artist,
				sort_title, sort_album_name, sort_artist_name, sort_album_artist_name,
				disc_subtitle, search_participants)
			VALUES (NEW.rowid, NEW.title, NEW.album, NEW.artist, NEW.album_artist,
				NEW.sort_title, NEW.sort_album_name, NEW.sort_artist_name, NEW.sort_album_artist_name,
				COALESCE(NEW.disc_subtitle, ''), COALESCE(NEW.search_participants, ''));
		END
	`)
	if err != nil {
		return fmt.Errorf("creating media_file_fts insert trigger: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE TRIGGER media_file_fts_ad AFTER DELETE ON media_file BEGIN
			INSERT INTO media_file_fts(media_file_fts, rowid, title, album, artist, album_artist,
				sort_title, sort_album_name, sort_artist_name, sort_album_artist_name,
				disc_subtitle, search_participants)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.album, OLD.artist, OLD.album_artist,
				OLD.sort_title, OLD.sort_album_name, OLD.sort_artist_name, OLD.sort_album_artist_name,
				COALESCE(OLD.disc_subtitle, ''), COALESCE(OLD.search_participants, ''));
		END
	`)
	if err != nil {
		return fmt.Errorf("creating media_file_fts delete trigger: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE TRIGGER media_file_fts_au AFTER UPDATE ON media_file BEGIN
			INSERT INTO media_file_fts(media_file_fts, rowid, title, album, artist, album_artist,
				sort_title, sort_album_name, sort_artist_name, sort_album_artist_name,
				disc_subtitle, search_participants)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.album, OLD.artist, OLD.album_artist,
				OLD.sort_title, OLD.sort_album_name, OLD.sort_artist_name, OLD.sort_album_artist_name,
				COALESCE(OLD.disc_subtitle, ''), COALESCE(OLD.search_participants, ''));
			INSERT INTO media_file_fts(rowid, title, album, artist, album_artist,
				sort_title, sort_album_name, sort_artist_name, sort_album_artist_name,
				disc_subtitle, search_participants)
			VALUES (NEW.rowid, NEW.title, NEW.album, NEW.artist, NEW.album_artist,
				NEW.sort_title, NEW.sort_album_name, NEW.sort_artist_name, NEW.sort_album_artist_name,
				COALESCE(NEW.disc_subtitle, ''), COALESCE(NEW.search_participants, ''));
		END
	`)
	if err != nil {
		return fmt.Errorf("creating media_file_fts update trigger: %w", err)
	}

	// Step 6: Create triggers for album
	_, err = tx.ExecContext(ctx, `
		CREATE TRIGGER album_fts_ai AFTER INSERT ON album BEGIN
			INSERT INTO album_fts(rowid, name, sort_album_name, album_artist,
				search_participants, discs, catalog_num, album_version)
			VALUES (NEW.rowid, NEW.name, COALESCE(NEW.sort_album_name, ''), COALESCE(NEW.album_artist, ''),
				COALESCE(NEW.search_participants, ''), COALESCE(NEW.discs, ''),
				COALESCE(NEW.catalog_num, ''), COALESCE(NEW.album_version, ''));
		END
	`)
	if err != nil {
		return fmt.Errorf("creating album_fts insert trigger: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE TRIGGER album_fts_ad AFTER DELETE ON album BEGIN
			INSERT INTO album_fts(album_fts, rowid, name, sort_album_name, album_artist,
				search_participants, discs, catalog_num, album_version)
			VALUES ('delete', OLD.rowid, OLD.name, COALESCE(OLD.sort_album_name, ''), COALESCE(OLD.album_artist, ''),
				COALESCE(OLD.search_participants, ''), COALESCE(OLD.discs, ''),
				COALESCE(OLD.catalog_num, ''), COALESCE(OLD.album_version, ''));
		END
	`)
	if err != nil {
		return fmt.Errorf("creating album_fts delete trigger: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE TRIGGER album_fts_au AFTER UPDATE ON album BEGIN
			INSERT INTO album_fts(album_fts, rowid, name, sort_album_name, album_artist,
				search_participants, discs, catalog_num, album_version)
			VALUES ('delete', OLD.rowid, OLD.name, COALESCE(OLD.sort_album_name, ''), COALESCE(OLD.album_artist, ''),
				COALESCE(OLD.search_participants, ''), COALESCE(OLD.discs, ''),
				COALESCE(OLD.catalog_num, ''), COALESCE(OLD.album_version, ''));
			INSERT INTO album_fts(rowid, name, sort_album_name, album_artist,
				search_participants, discs, catalog_num, album_version)
			VALUES (NEW.rowid, NEW.name, COALESCE(NEW.sort_album_name, ''), COALESCE(NEW.album_artist, ''),
				COALESCE(NEW.search_participants, ''), COALESCE(NEW.discs, ''),
				COALESCE(NEW.catalog_num, ''), COALESCE(NEW.album_version, ''));
		END
	`)
	if err != nil {
		return fmt.Errorf("creating album_fts update trigger: %w", err)
	}

	// Step 7: Create triggers for artist
	_, err = tx.ExecContext(ctx, `
		CREATE TRIGGER artist_fts_ai AFTER INSERT ON artist BEGIN
			INSERT INTO artist_fts(rowid, name, sort_artist_name)
			VALUES (NEW.rowid, NEW.name, COALESCE(NEW.sort_artist_name, ''));
		END
	`)
	if err != nil {
		return fmt.Errorf("creating artist_fts insert trigger: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE TRIGGER artist_fts_ad AFTER DELETE ON artist BEGIN
			INSERT INTO artist_fts(artist_fts, rowid, name, sort_artist_name)
			VALUES ('delete', OLD.rowid, OLD.name, COALESCE(OLD.sort_artist_name, ''));
		END
	`)
	if err != nil {
		return fmt.Errorf("creating artist_fts delete trigger: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE TRIGGER artist_fts_au AFTER UPDATE ON artist BEGIN
			INSERT INTO artist_fts(artist_fts, rowid, name, sort_artist_name)
			VALUES ('delete', OLD.rowid, OLD.name, COALESCE(OLD.sort_artist_name, ''));
			INSERT INTO artist_fts(rowid, name, sort_artist_name)
			VALUES (NEW.rowid, NEW.name, COALESCE(NEW.sort_artist_name, ''));
		END
	`)
	if err != nil {
		return fmt.Errorf("creating artist_fts update trigger: %w", err)
	}

	return nil
}

func downAddFts5Search(ctx context.Context, tx *sql.Tx) error {
	// Drop triggers
	for _, trigger := range []string{
		"media_file_fts_ai", "media_file_fts_ad", "media_file_fts_au",
		"album_fts_ai", "album_fts_ad", "album_fts_au",
		"artist_fts_ai", "artist_fts_ad", "artist_fts_au",
	} {
		_, err := tx.ExecContext(ctx, "DROP TRIGGER IF EXISTS "+trigger)
		if err != nil {
			return fmt.Errorf("dropping trigger %s: %w", trigger, err)
		}
	}

	// Drop FTS5 virtual tables
	for _, table := range []string{"media_file_fts", "album_fts", "artist_fts"} {
		_, err := tx.ExecContext(ctx, "DROP TABLE IF EXISTS "+table)
		if err != nil {
			return fmt.Errorf("dropping table %s: %w", table, err)
		}
	}

	// Note: We don't drop search_participants columns because SQLite doesn't support DROP COLUMN
	// on older versions, and the column is harmless if left in place.
	return nil
}

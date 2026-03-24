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

// stripPunct generates a SQL expression that strips common punctuation from a column or expression.
// Used during migration to approximate the Go normalizeForFTS function for bulk-populating search_normalized.
func stripPunct(col string) string {
	return fmt.Sprintf(
		`REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(%s, '.', ''), '/', ''), '-', ''), '''', ''), '&', ''), ',', '')`,
		col,
	)
}

func upAddFts5Search(ctx context.Context, tx *sql.Tx) error {
	notice(tx, "Adding FTS5 full-text search indexes. This may take a moment on large libraries.")

	// Step 1: Add search_participants and search_normalized columns to media_file, album, and artist
	_, err := tx.ExecContext(ctx, `ALTER TABLE media_file ADD COLUMN search_participants TEXT NOT NULL DEFAULT ''`)
	if err != nil {
		return fmt.Errorf("adding search_participants to media_file: %w", err)
	}
	_, err = tx.ExecContext(ctx, `ALTER TABLE media_file ADD COLUMN search_normalized TEXT NOT NULL DEFAULT ''`)
	if err != nil {
		return fmt.Errorf("adding search_normalized to media_file: %w", err)
	}
	_, err = tx.ExecContext(ctx, `ALTER TABLE album ADD COLUMN search_participants TEXT NOT NULL DEFAULT ''`)
	if err != nil {
		return fmt.Errorf("adding search_participants to album: %w", err)
	}
	_, err = tx.ExecContext(ctx, `ALTER TABLE album ADD COLUMN search_normalized TEXT NOT NULL DEFAULT ''`)
	if err != nil {
		return fmt.Errorf("adding search_normalized to album: %w", err)
	}
	_, err = tx.ExecContext(ctx, `ALTER TABLE artist ADD COLUMN search_normalized TEXT NOT NULL DEFAULT ''`)
	if err != nil {
		return fmt.Errorf("adding search_normalized to artist: %w", err)
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

	// Step 2b: Populate search_normalized using SQL REPLACE chains for common punctuation.
	// The Go code will compute the precise value on next scan; this is a best-effort approximation.
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		UPDATE artist SET search_normalized = %s
		WHERE name != %s`,
		stripPunct("name"), stripPunct("name")))
	if err != nil {
		return fmt.Errorf("populating artist search_normalized: %w", err)
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		UPDATE album SET search_normalized = TRIM(%s || ' ' || %s)
		WHERE name != %s OR COALESCE(album_artist, '') != %s`,
		stripPunct("name"), stripPunct("COALESCE(album_artist, '')"),
		stripPunct("name"), stripPunct("COALESCE(album_artist, '')")))
	if err != nil {
		return fmt.Errorf("populating album search_normalized: %w", err)
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		UPDATE media_file SET search_normalized =
			TRIM(%s || ' ' || %s || ' ' || %s || ' ' || %s)
		WHERE title != %s
			OR COALESCE(album, '') != %s
			OR COALESCE(artist, '') != %s
			OR COALESCE(album_artist, '') != %s`,
		stripPunct("title"), stripPunct("COALESCE(album, '')"),
		stripPunct("COALESCE(artist, '')"), stripPunct("COALESCE(album_artist, '')"),
		stripPunct("title"), stripPunct("COALESCE(album, '')"),
		stripPunct("COALESCE(artist, '')"), stripPunct("COALESCE(album_artist, '')")))
	if err != nil {
		return fmt.Errorf("populating media_file search_normalized: %w", err)
	}

	// Step 3: Create FTS5 virtual tables
	_, err = tx.ExecContext(ctx, `
		CREATE VIRTUAL TABLE IF NOT EXISTS media_file_fts USING fts5(
			title, album, artist, album_artist,
			sort_title, sort_album_name, sort_artist_name, sort_album_artist_name,
			disc_subtitle, search_participants, search_normalized,
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
			search_participants, discs, catalog_num, album_version, search_normalized,
			content='', content_rowid='rowid',
			tokenize='unicode61 remove_diacritics 2'
		)
	`)
	if err != nil {
		return fmt.Errorf("creating album_fts: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE VIRTUAL TABLE IF NOT EXISTS artist_fts USING fts5(
			name, sort_artist_name, search_normalized,
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
			disc_subtitle, search_participants, search_normalized)
		SELECT rowid, title, album, artist, album_artist,
			sort_title, sort_album_name, sort_artist_name, sort_album_artist_name,
			COALESCE(disc_subtitle, ''), COALESCE(search_participants, ''),
			COALESCE(search_normalized, '')
		FROM media_file
	`)
	if err != nil {
		return fmt.Errorf("populating media_file_fts: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO album_fts(rowid, name, sort_album_name, album_artist,
			search_participants, discs, catalog_num, album_version, search_normalized)
		SELECT rowid, name, COALESCE(sort_album_name, ''), COALESCE(album_artist, ''),
			COALESCE(search_participants, ''), COALESCE(discs, ''),
			COALESCE(catalog_num, ''),
			COALESCE((SELECT group_concat(json_extract(je.value, '$.value'), ' ')
				FROM json_each(album.tags, '$.albumversion') AS je), ''),
			COALESCE(search_normalized, '')
		FROM album
	`)
	if err != nil {
		return fmt.Errorf("populating album_fts: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO artist_fts(rowid, name, sort_artist_name, search_normalized)
		SELECT rowid, name, COALESCE(sort_artist_name, ''), COALESCE(search_normalized, '')
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
				disc_subtitle, search_participants, search_normalized)
			VALUES (NEW.rowid, NEW.title, NEW.album, NEW.artist, NEW.album_artist,
				NEW.sort_title, NEW.sort_album_name, NEW.sort_artist_name, NEW.sort_album_artist_name,
				COALESCE(NEW.disc_subtitle, ''), COALESCE(NEW.search_participants, ''),
				COALESCE(NEW.search_normalized, ''));
		END
	`)
	if err != nil {
		return fmt.Errorf("creating media_file_fts insert trigger: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE TRIGGER media_file_fts_ad AFTER DELETE ON media_file BEGIN
			INSERT INTO media_file_fts(media_file_fts, rowid, title, album, artist, album_artist,
				sort_title, sort_album_name, sort_artist_name, sort_album_artist_name,
				disc_subtitle, search_participants, search_normalized)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.album, OLD.artist, OLD.album_artist,
				OLD.sort_title, OLD.sort_album_name, OLD.sort_artist_name, OLD.sort_album_artist_name,
				COALESCE(OLD.disc_subtitle, ''), COALESCE(OLD.search_participants, ''),
				COALESCE(OLD.search_normalized, ''));
		END
	`)
	if err != nil {
		return fmt.Errorf("creating media_file_fts delete trigger: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE TRIGGER media_file_fts_au AFTER UPDATE ON media_file
		WHEN
			OLD.title IS NOT NEW.title OR
			OLD.album IS NOT NEW.album OR
			OLD.artist IS NOT NEW.artist OR
			OLD.album_artist IS NOT NEW.album_artist OR
			OLD.sort_title IS NOT NEW.sort_title OR
			OLD.sort_album_name IS NOT NEW.sort_album_name OR
			OLD.sort_artist_name IS NOT NEW.sort_artist_name OR
			OLD.sort_album_artist_name IS NOT NEW.sort_album_artist_name OR
			OLD.disc_subtitle IS NOT NEW.disc_subtitle OR
			OLD.search_participants IS NOT NEW.search_participants OR
			OLD.search_normalized IS NOT NEW.search_normalized
		BEGIN
			INSERT INTO media_file_fts(media_file_fts, rowid, title, album, artist, album_artist,
				sort_title, sort_album_name, sort_artist_name, sort_album_artist_name,
				disc_subtitle, search_participants, search_normalized)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.album, OLD.artist, OLD.album_artist,
				OLD.sort_title, OLD.sort_album_name, OLD.sort_artist_name, OLD.sort_album_artist_name,
				COALESCE(OLD.disc_subtitle, ''), COALESCE(OLD.search_participants, ''),
				COALESCE(OLD.search_normalized, ''));
			INSERT INTO media_file_fts(rowid, title, album, artist, album_artist,
				sort_title, sort_album_name, sort_artist_name, sort_album_artist_name,
				disc_subtitle, search_participants, search_normalized)
			VALUES (NEW.rowid, NEW.title, NEW.album, NEW.artist, NEW.album_artist,
				NEW.sort_title, NEW.sort_album_name, NEW.sort_artist_name, NEW.sort_album_artist_name,
				COALESCE(NEW.disc_subtitle, ''), COALESCE(NEW.search_participants, ''),
				COALESCE(NEW.search_normalized, ''));
		END
	`)
	if err != nil {
		return fmt.Errorf("creating media_file_fts update trigger: %w", err)
	}

	// Step 6: Create triggers for album
	_, err = tx.ExecContext(ctx, `
		CREATE TRIGGER album_fts_ai AFTER INSERT ON album BEGIN
			INSERT INTO album_fts(rowid, name, sort_album_name, album_artist,
				search_participants, discs, catalog_num, album_version, search_normalized)
			VALUES (NEW.rowid, NEW.name, COALESCE(NEW.sort_album_name, ''), COALESCE(NEW.album_artist, ''),
				COALESCE(NEW.search_participants, ''), COALESCE(NEW.discs, ''),
				COALESCE(NEW.catalog_num, ''),
				COALESCE((SELECT group_concat(json_extract(je.value, '$.value'), ' ')
					FROM json_each(NEW.tags, '$.albumversion') AS je), ''),
				COALESCE(NEW.search_normalized, ''));
		END
	`)
	if err != nil {
		return fmt.Errorf("creating album_fts insert trigger: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE TRIGGER album_fts_ad AFTER DELETE ON album BEGIN
			INSERT INTO album_fts(album_fts, rowid, name, sort_album_name, album_artist,
				search_participants, discs, catalog_num, album_version, search_normalized)
			VALUES ('delete', OLD.rowid, OLD.name, COALESCE(OLD.sort_album_name, ''), COALESCE(OLD.album_artist, ''),
				COALESCE(OLD.search_participants, ''), COALESCE(OLD.discs, ''),
				COALESCE(OLD.catalog_num, ''),
				COALESCE((SELECT group_concat(json_extract(je.value, '$.value'), ' ')
					FROM json_each(OLD.tags, '$.albumversion') AS je), ''),
				COALESCE(OLD.search_normalized, ''));
		END
	`)
	if err != nil {
		return fmt.Errorf("creating album_fts delete trigger: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE TRIGGER album_fts_au AFTER UPDATE ON album
		WHEN
			OLD.name IS NOT NEW.name OR
			OLD.sort_album_name IS NOT NEW.sort_album_name OR
			OLD.album_artist IS NOT NEW.album_artist OR
			OLD.search_participants IS NOT NEW.search_participants OR
			OLD.discs IS NOT NEW.discs OR
			OLD.catalog_num IS NOT NEW.catalog_num OR
			OLD.tags IS NOT NEW.tags OR
			OLD.search_normalized IS NOT NEW.search_normalized
		BEGIN
			INSERT INTO album_fts(album_fts, rowid, name, sort_album_name, album_artist,
				search_participants, discs, catalog_num, album_version, search_normalized)
			VALUES ('delete', OLD.rowid, OLD.name, COALESCE(OLD.sort_album_name, ''), COALESCE(OLD.album_artist, ''),
				COALESCE(OLD.search_participants, ''), COALESCE(OLD.discs, ''),
				COALESCE(OLD.catalog_num, ''),
				COALESCE((SELECT group_concat(json_extract(je.value, '$.value'), ' ')
					FROM json_each(OLD.tags, '$.albumversion') AS je), ''),
				COALESCE(OLD.search_normalized, ''));
			INSERT INTO album_fts(rowid, name, sort_album_name, album_artist,
				search_participants, discs, catalog_num, album_version, search_normalized)
			VALUES (NEW.rowid, NEW.name, COALESCE(NEW.sort_album_name, ''), COALESCE(NEW.album_artist, ''),
				COALESCE(NEW.search_participants, ''), COALESCE(NEW.discs, ''),
				COALESCE(NEW.catalog_num, ''),
				COALESCE((SELECT group_concat(json_extract(je.value, '$.value'), ' ')
					FROM json_each(NEW.tags, '$.albumversion') AS je), ''),
				COALESCE(NEW.search_normalized, ''));
		END
	`)
	if err != nil {
		return fmt.Errorf("creating album_fts update trigger: %w", err)
	}

	// Step 7: Create triggers for artist
	_, err = tx.ExecContext(ctx, `
		CREATE TRIGGER artist_fts_ai AFTER INSERT ON artist BEGIN
			INSERT INTO artist_fts(rowid, name, sort_artist_name, search_normalized)
			VALUES (NEW.rowid, NEW.name, COALESCE(NEW.sort_artist_name, ''),
				COALESCE(NEW.search_normalized, ''));
		END
	`)
	if err != nil {
		return fmt.Errorf("creating artist_fts insert trigger: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE TRIGGER artist_fts_ad AFTER DELETE ON artist BEGIN
			INSERT INTO artist_fts(artist_fts, rowid, name, sort_artist_name, search_normalized)
			VALUES ('delete', OLD.rowid, OLD.name, COALESCE(OLD.sort_artist_name, ''),
				COALESCE(OLD.search_normalized, ''));
		END
	`)
	if err != nil {
		return fmt.Errorf("creating artist_fts delete trigger: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE TRIGGER artist_fts_au AFTER UPDATE ON artist
		WHEN
			OLD.name IS NOT NEW.name OR
			OLD.sort_artist_name IS NOT NEW.sort_artist_name OR
			OLD.search_normalized IS NOT NEW.search_normalized
		BEGIN
			INSERT INTO artist_fts(artist_fts, rowid, name, sort_artist_name, search_normalized)
			VALUES ('delete', OLD.rowid, OLD.name, COALESCE(OLD.sort_artist_name, ''),
				COALESCE(OLD.search_normalized, ''));
			INSERT INTO artist_fts(rowid, name, sort_artist_name, search_normalized)
			VALUES (NEW.rowid, NEW.name, COALESCE(NEW.sort_artist_name, ''),
				COALESCE(NEW.search_normalized, ''));
		END
	`)
	if err != nil {
		return fmt.Errorf("creating artist_fts update trigger: %w", err)
	}

	return nil
}

func downAddFts5Search(ctx context.Context, tx *sql.Tx) error {
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

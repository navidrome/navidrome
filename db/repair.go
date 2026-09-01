package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
)

var ftsTables = []string{"media_file_fts", "album_fts", "artist_fts"}

var ftsTriggerSuffixes = []string{"_ai", "_ad", "_au"}

// IntegrityCheck runs PRAGMA integrity_check and returns the reported problems,
// or an empty slice when the database is healthy.
func IntegrityCheck(ctx context.Context, database *sql.DB) ([]string, error) {
	rows, err := database.QueryContext(ctx, "PRAGMA integrity_check")
	if err != nil {
		return nil, fmt.Errorf("running integrity_check: %w", err)
	}
	defer rows.Close()

	var issues []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return nil, fmt.Errorf("reading integrity_check results: %w", err)
		}
		issues = append(issues, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading integrity_check results: %w", err)
	}
	if len(issues) == 1 && issues[0] == "ok" {
		return nil, nil
	}
	return issues, nil
}

// ForeignKeyCheck runs PRAGMA foreign_key_check and returns one line per row that
// references a missing parent, or an empty slice when there are none.
func ForeignKeyCheck(ctx context.Context, database *sql.DB) ([]string, error) {
	rows, err := database.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return nil, fmt.Errorf("running foreign_key_check: %w", err)
	}
	defer rows.Close()

	var violations []string
	for rows.Next() {
		var table, parent string
		var rowid sql.NullInt64
		var fkid int
		if err := rows.Scan(&table, &rowid, &parent, &fkid); err != nil {
			return nil, fmt.Errorf("reading foreign_key_check results: %w", err)
		}
		violations = append(violations,
			fmt.Sprintf("%s (rowid %d) references a missing row in %s", table, rowid.Int64, parent))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading foreign_key_check results: %w", err)
	}
	return violations, nil
}

// IsFTSCorruptionOnly reports whether every integrity issue refers to one of the
// FTS5 search tables, meaning RebuildFTS can fully repair the database.
func IsFTSCorruptionOnly(issues []string) bool {
	if len(issues) == 0 {
		return false
	}
	for _, line := range issues {
		if !slices.ContainsFunc(ftsTables, func(table string) bool { return strings.Contains(line, table) }) {
			return false
		}
	}
	return true
}

// VerifyFTS runs the FTS5 'integrity-check' command on each search table. Unlike a
// full PRAGMA integrity_check, it reads only the FTS indexes, not the whole database.
func VerifyFTS(ctx context.Context, database *sql.DB) error {
	for _, table := range ftsTables {
		stmt := fmt.Sprintf("INSERT INTO %[1]s(%[1]s) VALUES('integrity-check')", table) //nolint:gosec // fixed table list
		if _, err := database.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("verifying %s: %w", table, err)
		}
	}
	return nil
}

const ftsSearchMigration int64 = 20260220173400

// RebuildFTS drops the FTS5 search tables and their triggers, recreates them, and
// repopulates the indexes from the base tables. The FTS tables are contentless, so
// no user data is lost.
// It only requires the FTS migration, not a fully migrated schema: a corrupted DB
// often cannot run pending migrations, and the rebuild is transactional, so a column
// mismatch with a newer schema fails loudly and rolls back.
func RebuildFTS(ctx context.Context, database *sql.DB) error {
	var applied int
	err := database.QueryRowContext(ctx,
		"SELECT count(*) FROM goose_db_version WHERE version_id = ?", ftsSearchMigration).Scan(&applied)
	if err != nil {
		return fmt.Errorf("checking FTS migration status: %w", err)
	}
	if applied == 0 {
		return errors.New("the FTS search migration has not been applied yet; start Navidrome once to migrate the database first")
	}

	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting FTS rebuild transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var stmts []string
	for _, table := range ftsTables {
		for _, suffix := range ftsTriggerSuffixes {
			stmts = append(stmts, "DROP TRIGGER IF EXISTS "+table+suffix)
		}
		stmts = append(stmts, "DROP TABLE IF EXISTS "+table)
	}
	stmts = append(stmts, ftsSchemaDDL...)
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("rebuilding FTS schema: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing FTS rebuild: %w", err)
	}
	return nil
}

// ftsSchemaDDL must stay identical to the add_fts5_search migration; the schema
// comparison in repair_test.go guards against drift.
var ftsSchemaDDL = []string{
	`
		CREATE VIRTUAL TABLE IF NOT EXISTS media_file_fts USING fts5(
			title, album, artist, album_artist,
			sort_title, sort_album_name, sort_artist_name, sort_album_artist_name,
			disc_subtitle, search_participants, search_normalized,
			content='', content_rowid='rowid',
			tokenize='unicode61 remove_diacritics 2'
		)
	`,
	`
		CREATE VIRTUAL TABLE IF NOT EXISTS album_fts USING fts5(
			name, sort_album_name, album_artist,
			search_participants, discs, catalog_num, album_version, search_normalized,
			content='', content_rowid='rowid',
			tokenize='unicode61 remove_diacritics 2'
		)
	`,
	`
		CREATE VIRTUAL TABLE IF NOT EXISTS artist_fts USING fts5(
			name, sort_artist_name, search_normalized,
			content='', content_rowid='rowid',
			tokenize='unicode61 remove_diacritics 2'
		)
	`,
	`
		INSERT INTO media_file_fts(rowid, title, album, artist, album_artist,
			sort_title, sort_album_name, sort_artist_name, sort_album_artist_name,
			disc_subtitle, search_participants, search_normalized)
		SELECT rowid, title, album, artist, album_artist,
			sort_title, sort_album_name, sort_artist_name, sort_album_artist_name,
			COALESCE(disc_subtitle, ''), COALESCE(search_participants, ''),
			COALESCE(search_normalized, '')
		FROM media_file
	`,
	`
		INSERT INTO album_fts(rowid, name, sort_album_name, album_artist,
			search_participants, discs, catalog_num, album_version, search_normalized)
		SELECT rowid, name, COALESCE(sort_album_name, ''), COALESCE(album_artist, ''),
			COALESCE(search_participants, ''), COALESCE(discs, ''),
			COALESCE(catalog_num, ''),
			COALESCE((SELECT group_concat(json_extract(je.value, '$.value'), ' ')
				FROM json_each(album.tags, '$.albumversion') AS je), ''),
			COALESCE(search_normalized, '')
		FROM album
	`,
	`
		INSERT INTO artist_fts(rowid, name, sort_artist_name, search_normalized)
		SELECT rowid, name, COALESCE(sort_artist_name, ''), COALESCE(search_normalized, '')
		FROM artist
	`,
	`
		CREATE TRIGGER media_file_fts_ai AFTER INSERT ON media_file BEGIN
			INSERT INTO media_file_fts(rowid, title, album, artist, album_artist,
				sort_title, sort_album_name, sort_artist_name, sort_album_artist_name,
				disc_subtitle, search_participants, search_normalized)
			VALUES (NEW.rowid, NEW.title, NEW.album, NEW.artist, NEW.album_artist,
				NEW.sort_title, NEW.sort_album_name, NEW.sort_artist_name, NEW.sort_album_artist_name,
				COALESCE(NEW.disc_subtitle, ''), COALESCE(NEW.search_participants, ''),
				COALESCE(NEW.search_normalized, ''));
		END
	`,
	`
		CREATE TRIGGER media_file_fts_ad AFTER DELETE ON media_file BEGIN
			INSERT INTO media_file_fts(media_file_fts, rowid, title, album, artist, album_artist,
				sort_title, sort_album_name, sort_artist_name, sort_album_artist_name,
				disc_subtitle, search_participants, search_normalized)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.album, OLD.artist, OLD.album_artist,
				OLD.sort_title, OLD.sort_album_name, OLD.sort_artist_name, OLD.sort_album_artist_name,
				COALESCE(OLD.disc_subtitle, ''), COALESCE(OLD.search_participants, ''),
				COALESCE(OLD.search_normalized, ''));
		END
	`,
	`
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
	`,
	`
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
	`,
	`
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
	`,
	`
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
	`,
	`
		CREATE TRIGGER artist_fts_ai AFTER INSERT ON artist BEGIN
			INSERT INTO artist_fts(rowid, name, sort_artist_name, search_normalized)
			VALUES (NEW.rowid, NEW.name, COALESCE(NEW.sort_artist_name, ''),
				COALESCE(NEW.search_normalized, ''));
		END
	`,
	`
		CREATE TRIGGER artist_fts_ad AFTER DELETE ON artist BEGIN
			INSERT INTO artist_fts(artist_fts, rowid, name, sort_artist_name, search_normalized)
			VALUES ('delete', OLD.rowid, OLD.name, COALESCE(OLD.sort_artist_name, ''),
				COALESCE(OLD.search_normalized, ''));
		END
	`,
	`
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
	`,
}

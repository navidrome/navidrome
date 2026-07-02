package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/navidrome/navidrome/utils/str"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upBackfillArtistSearchNormalized, downBackfillArtistSearchNormalized)
}

// The FTS5 migration back-filled artist.search_normalized with a SQL approximation that
// cannot transliterate atomic letters (Ø, æ, ß, ...), and the scanner never rewrote the
// column, leaving artists like "GØGGS" unfindable by ASCII searches. Recompute it in Go;
// the artist_fts update trigger re-indexes every row that changes.
func upBackfillArtistSearchNormalized(ctx context.Context, tx *sql.Tx) error {
	notice(ctx, tx, "Rebuilding artist search index data. This may take a moment on large libraries.")

	rows, err := tx.QueryContext(ctx, "SELECT id, name, search_normalized FROM artist")
	if err != nil {
		return fmt.Errorf("querying artists: %w", err)
	}
	defer rows.Close()

	updates := map[string]string{}
	for rows.Next() {
		var id, name, current string
		if err := rows.Scan(&id, &name, &current); err != nil {
			return fmt.Errorf("scanning artist: %w", err)
		}
		if expected := str.NormalizeForFTS(name); expected != current {
			updates[id] = expected
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating artists: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, "UPDATE artist SET search_normalized = ? WHERE id = ?")
	if err != nil {
		return fmt.Errorf("preparing update: %w", err)
	}
	defer stmt.Close()
	for id, normalized := range updates {
		if _, err := stmt.ExecContext(ctx, normalized, id); err != nil {
			return fmt.Errorf("updating artist %s: %w", id, err)
		}
	}
	return nil
}

func downBackfillArtistSearchNormalized(context.Context, *sql.Tx) error {
	return nil
}

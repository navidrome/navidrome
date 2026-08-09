package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddAlbumReplaygain, downAddAlbumReplaygain)
}

func upAddAlbumReplaygain(ctx context.Context, tx *sql.Tx) error {
	// Backfill the most-frequent value per album (matching MediaFiles.ToAlbum), staging RG-bearing rows
	// into an indexed temp table — a correlated subquery over a windowed CTE re-scans media_file per album.
	_, err := tx.ExecContext(ctx, `
ALTER TABLE album ADD COLUMN rg_album_gain real;
ALTER TABLE album ADD COLUMN rg_album_peak real;

CREATE TEMP TABLE _rg_backfill AS
	SELECT album_id, rg_album_gain, rg_album_peak FROM media_file
	WHERE rg_album_gain IS NOT NULL OR rg_album_peak IS NOT NULL;
CREATE INDEX _rg_backfill_album ON _rg_backfill(album_id);

UPDATE album SET
	rg_album_gain = (SELECT rg_album_gain FROM _rg_backfill WHERE _rg_backfill.album_id = album.id AND rg_album_gain IS NOT NULL
		GROUP BY rg_album_gain ORDER BY count(*) DESC, rg_album_gain LIMIT 1),
	rg_album_peak = (SELECT rg_album_peak FROM _rg_backfill WHERE _rg_backfill.album_id = album.id AND rg_album_peak IS NOT NULL
		GROUP BY rg_album_peak ORDER BY count(*) DESC, rg_album_peak LIMIT 1)
WHERE album.id IN (SELECT album_id FROM _rg_backfill);

DROP TABLE _rg_backfill;
	`)
	return err
}

func downAddAlbumReplaygain(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}

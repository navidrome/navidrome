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
	// Backfill the most-frequent non-null value per album, matching MediaFiles.ToAlbum. A plain
	// max() would pick a minority outlier, and GetTouchedAlbums never re-derives an unchanged album,
	// so a wrong value would persist. The CTEs group media_file once instead of scanning it per album.
	_, err := tx.ExecContext(ctx, `
ALTER TABLE album ADD COLUMN rg_album_gain real;
ALTER TABLE album ADD COLUMN rg_album_peak real;

WITH gain_mode AS (
	SELECT album_id, rg_album_gain AS val,
		row_number() OVER (PARTITION BY album_id ORDER BY count(*) DESC, rg_album_gain) AS rn
	FROM media_file WHERE rg_album_gain IS NOT NULL
	GROUP BY album_id, rg_album_gain
),
peak_mode AS (
	SELECT album_id, rg_album_peak AS val,
		row_number() OVER (PARTITION BY album_id ORDER BY count(*) DESC, rg_album_peak) AS rn
	FROM media_file WHERE rg_album_peak IS NOT NULL
	GROUP BY album_id, rg_album_peak
)
UPDATE album SET
	rg_album_gain = (SELECT val FROM gain_mode WHERE gain_mode.album_id = album.id AND gain_mode.rn = 1),
	rg_album_peak = (SELECT val FROM peak_mode WHERE peak_mode.album_id = album.id AND peak_mode.rn = 1);
	`)
	return err
}

func downAddAlbumReplaygain(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}

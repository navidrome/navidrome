package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upMakeReplaygainFieldsNullable, downMakeReplaygainFieldsNullable)
}

func upMakeReplaygainFieldsNullable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
ALTER TABLE media_file ADD COLUMN rg_album_gain_new real;
ALTER TABLE media_file ADD COLUMN rg_album_peak_new real;
ALTER TABLE media_file ADD COLUMN rg_track_gain_new real;
ALTER TABLE media_file ADD COLUMN rg_track_peak_new real;

UPDATE media_file SET
	rg_album_gain_new = rg_album_gain,
	rg_album_peak_new = rg_album_peak,
	rg_track_gain_new = rg_track_gain,
	rg_track_peak_new = rg_track_peak;

ALTER TABLE media_file DROP COLUMN rg_album_gain;
ALTER TABLE media_file DROP COLUMN rg_album_peak;
ALTER TABLE media_file DROP COLUMN rg_track_gain;
ALTER TABLE media_file DROP COLUMN rg_track_peak;

ALTER TABLE media_file RENAME COLUMN rg_album_gain_new TO rg_album_gain;
ALTER TABLE media_file RENAME COLUMN rg_album_peak_new TO rg_album_peak;
ALTER TABLE media_file RENAME COLUMN rg_track_gain_new TO rg_track_gain;
ALTER TABLE media_file RENAME COLUMN rg_track_peak_new TO rg_track_peak;
	`)

	if err != nil {
		return err
	}

	notice(tx, "Fetching replaygain fields properly will require a full scan")
	return nil
}

func downMakeReplaygainFieldsNullable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}

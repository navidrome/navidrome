package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upSupportMediaWithSubtracks, downSupportMediaWithSubtracks)
}

func upSupportMediaWithSubtracks(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		-- Add offset and sub_track columns
		ALTER TABLE media_file ADD COLUMN offset real DEFAULT 0 NOT NULL;
		ALTER TABLE media_file ADD COLUMN sub_track integer DEFAULT -1 NOT NULL;
		ALTER TABLE media_file ADD COLUMN cuefile varchar(255) DEFAULT '' NOT NULL;
	`)
	if err != nil {
		return err
	}
	notice(tx, "A full scan will be triggered to populate the new tables. This may take a while.")
	return forceFullRescan(tx)
}

func downSupportMediaWithSubtracks(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		-- Remove data for sub tracks
		DELETE FROM media_file WHERE sub_track != -1;
		-- Remove offset and sub_track columns
		ALTER TABLE media_file DROP COLUMN offset DEFAULT 0 NOT NULL;
		ALTER TABLE media_file DROP COLUMN sub_track integer DEFAULT -1 NOT NULL;
		ALTER TABLE media_file DROP COLUMN cuefile varchar(255) DEFAULT '' NOT NULL;
`)
	return err
}

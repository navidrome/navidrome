package migrations

import (
	"context"
	"database/sql"

	"github.com/navidrome/navidrome/consts"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddPerLibraryPid, downAddPerLibraryPid)
}

func upAddPerLibraryPid(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		ALTER TABLE library ADD COLUMN pid_album TEXT NOT NULL DEFAULT '';
		ALTER TABLE library ADD COLUMN pid_track TEXT NOT NULL DEFAULT '';
		ALTER TABLE library ADD COLUMN scanned_pid_album TEXT NOT NULL DEFAULT '';
		ALTER TABLE library ADD COLUMN scanned_pid_track TEXT NOT NULL DEFAULT '';
	`)
	if err != nil {
		return err
	}

	// Seed from the global properties so existing installs don't full-scan on upgrade
	_, err = tx.ExecContext(ctx, `
		UPDATE library SET
			scanned_pid_album = COALESCE((SELECT value FROM property WHERE id = ?), ?),
			scanned_pid_track = COALESCE((SELECT value FROM property WHERE id = ?), ?);
	`, consts.PIDAlbumKey, consts.DefaultAlbumPID, consts.PIDTrackKey, consts.DefaultTrackPID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM property WHERE id IN (?, ?);`,
		consts.PIDAlbumKey, consts.PIDTrackKey)
	return err
}

func downAddPerLibraryPid(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		ALTER TABLE library DROP COLUMN pid_album;
		ALTER TABLE library DROP COLUMN pid_track;
		ALTER TABLE library DROP COLUMN scanned_pid_album;
		ALTER TABLE library DROP COLUMN scanned_pid_track;
	`)
	return err
}

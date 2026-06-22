package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upRenameMusicbrainzRecordingId, downRenameMusicbrainzRecordingId)
}

func upRenameMusicbrainzRecordingId(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
alter table media_file
	rename column mbz_track_id to mbz_recording_id;
`)
	return err
}

func downRenameMusicbrainzRecordingId(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
alter table media_file
	rename column mbz_recording_id to mbz_track_id;
`)
	return err
}

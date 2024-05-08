package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upRenameMusicbrainzRecordingId, downRenameMusicbrainzRecordingId)
}

func upRenameMusicbrainzRecordingId(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file
	rename column mbz_track_id to mbz_recording_id;
`)
	return err
}

func downRenameMusicbrainzRecordingId(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file
	rename column mbz_recording_id to mbz_track_id;
`)
	return err
}

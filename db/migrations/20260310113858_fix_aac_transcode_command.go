package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upFixAacTranscodeCommand, downFixAacTranscodeCommand)
}

func upFixAacTranscodeCommand(_ context.Context, tx *sql.Tx) error {
	// The old AAC command used `-f ipod -movflags frag_keyframe+empty_moov` which produces
	// corrupt/silent audio when ffmpeg pipes to stdout (confirmed in ffmpeg 8.0+).
	// Switch to `-f adts` (raw AAC framing) which works reliably via pipe.
	// Only update rows that still have the old default command.
	const oldCommand = "ffmpeg -i %s -ss %t -map 0:a:0 -b:a %bk -v 0 -c:a aac -f ipod -movflags frag_keyframe+empty_moov -"
	const newCommand = "ffmpeg -i %s -ss %t -map 0:a:0 -b:a %bk -v 0 -c:a aac -f adts -"
	_, err := tx.Exec(
		"UPDATE transcoding SET command = ? WHERE target_format = 'aac' AND command = ?",
		newCommand, oldCommand,
	)
	return err
}

func downFixAacTranscodeCommand(_ context.Context, tx *sql.Tx) error {
	return nil
}

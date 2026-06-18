package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddMetadataToDefaultTranscodings, downAddMetadataToDefaultTranscodings)
}

// metadataPairs maps the current default commands (no metadata mapping) to the
// new defaults that preserve source tags. Index 0 = old, index 1 = new.
//
// The new commands add `-map_metadata 0 -map_metadata 0:s:0` after `-map 0:a:0`:
// `-map_metadata 0` copies format-level tags (MP3/FLAC sources) and
// `-map_metadata 0:s:0` copies stream-level tags (OPUS/OGG sources); both are
// needed because the two source families store tags at different levels.
//
// AAC is included for consistency, but its `-f adts` container cannot hold tags,
// so the flags are a no-op there.
//
// Only rows still holding the exact unmodified default are updated, so any
// user-customized command is left untouched.
var metadataPairs = [][2]string{
	{
		"ffmpeg -ss %t -i %s -map 0:a:0 -b:a %bk -v 0 -f mp3 -",
		"ffmpeg -ss %t -i %s -map 0:a:0 -map_metadata 0 -map_metadata 0:s:0 -b:a %bk -v 0 -f mp3 -",
	},
	{
		"ffmpeg -ss %t -i %s -map 0:a:0 -b:a %bk -v 0 -c:a libopus -f opus -",
		"ffmpeg -ss %t -i %s -map 0:a:0 -map_metadata 0 -map_metadata 0:s:0 -b:a %bk -v 0 -c:a libopus -f opus -",
	},
	{
		"ffmpeg -ss %t -i %s -map 0:a:0 -b:a %bk -v 0 -c:a aac -f adts -",
		"ffmpeg -ss %t -i %s -map 0:a:0 -map_metadata 0 -map_metadata 0:s:0 -b:a %bk -v 0 -c:a aac -f adts -",
	},
	{
		"ffmpeg -ss %t -i %s -map 0:a:0 -v 0 -c:a flac -f flac -",
		"ffmpeg -ss %t -i %s -map 0:a:0 -map_metadata 0 -map_metadata 0:s:0 -v 0 -c:a flac -f flac -",
	},
}

func upAddMetadataToDefaultTranscodings(_ context.Context, tx *sql.Tx) error {
	for _, p := range metadataPairs {
		if _, err := tx.Exec(`UPDATE transcoding SET command = ? WHERE command = ?`, p[1], p[0]); err != nil {
			return err
		}
	}
	return nil
}

func downAddMetadataToDefaultTranscodings(_ context.Context, tx *sql.Tx) error {
	for _, p := range metadataPairs {
		if _, err := tx.Exec(`UPDATE transcoding SET command = ? WHERE command = ?`, p[0], p[1]); err != nil {
			return err
		}
	}
	return nil
}

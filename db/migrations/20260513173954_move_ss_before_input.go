package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upMoveSsBeforeInput, downMoveSsBeforeInput)
}

// ssSeekPairs maps old commands (output seeking) to new commands (input seeking).
// Index 0 = old (after -i), index 1 = new (before -i).
var ssSeekPairs = [][2]string{
	{
		"ffmpeg -i %s -ss %t -map 0:a:0 -b:a %bk -v 0 -f mp3 -",
		"ffmpeg -ss %t -i %s -map 0:a:0 -b:a %bk -v 0 -f mp3 -",
	},
	{
		"ffmpeg -i %s -ss %t -map 0:a:0 -b:a %bk -v 0 -c:a libopus -f opus -",
		"ffmpeg -ss %t -i %s -map 0:a:0 -b:a %bk -v 0 -c:a libopus -f opus -",
	},
	{
		"ffmpeg -i %s -ss %t -map 0:a:0 -b:a %bk -v 0 -c:a aac -f adts -",
		"ffmpeg -ss %t -i %s -map 0:a:0 -b:a %bk -v 0 -c:a aac -f adts -",
	},
	{
		"ffmpeg -i %s -ss %t -map 0:a:0 -b:a %bk -v 0 -c:a aac -f ipod -movflags frag_keyframe+empty_moov -",
		"ffmpeg -ss %t -i %s -map 0:a:0 -b:a %bk -v 0 -c:a aac -f ipod -movflags frag_keyframe+empty_moov -",
	},
	{
		"ffmpeg -i %s -ss %t -map 0:a:0 -v 0 -c:a flac -f flac -",
		"ffmpeg -ss %t -i %s -map 0:a:0 -v 0 -c:a flac -f flac -",
	},
}

func upMoveSsBeforeInput(_ context.Context, tx *sql.Tx) error {
	for _, p := range ssSeekPairs {
		if _, err := tx.Exec(`UPDATE transcoding SET command = ? WHERE command = ?`, p[1], p[0]); err != nil {
			return err
		}
	}
	return nil
}

func downMoveSsBeforeInput(_ context.Context, tx *sql.Tx) error {
	for _, p := range ssSeekPairs {
		if _, err := tx.Exec(`UPDATE transcoding SET command = ? WHERE command = ?`, p[0], p[1]); err != nil {
			return err
		}
	}
	return nil
}

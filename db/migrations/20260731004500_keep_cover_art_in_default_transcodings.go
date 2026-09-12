package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upKeepCoverArtInDefaultTranscodings, downKeepCoverArtInDefaultTranscodings)
}

// coverArtPairs maps the current default commands (audio stream only) to the new
// defaults that also carry over embedded cover art. Index 0 = old, index 1 = new.
//
// `-map 0:v:0?` picks up the attached picture when the source has one, and the
// trailing `?` keeps the command working for sources without artwork. `-c:v copy`
// avoids re-encoding it and `-disposition:v attached_pic` marks it as cover art
// rather than a video track.
//
// Only mp3 and flac are updated: the opus muxer rejects the mjpeg stream
// ("Unsupported codec id in stream 1") and adts refuses any video stream
// ("adts muxer does not support any stream of type video"), so adding the
// mapping there would break transcoding to those formats outright.
//
// Only rows still holding the exact unmodified default are updated, so any
// user-customized command is left untouched.
var coverArtPairs = [][2]string{
	{
		"ffmpeg -ss %t -i %s -map 0:a:0 -map_metadata 0 -map_metadata 0:s:a:0 -b:a %bk -v 0 -f mp3 -",
		"ffmpeg -ss %t -i %s -map 0:a:0 -map 0:v:0? -map_metadata 0 -map_metadata 0:s:a:0 -b:a %bk -v 0 -c:v copy -disposition:v attached_pic -f mp3 -",
	},
	{
		"ffmpeg -ss %t -i %s -map 0:a:0 -map_metadata 0 -map_metadata 0:s:a:0 -v 0 -c:a flac -f flac -",
		"ffmpeg -ss %t -i %s -map 0:a:0 -map 0:v:0? -map_metadata 0 -map_metadata 0:s:a:0 -v 0 -c:a flac -c:v copy -disposition:v attached_pic -f flac -",
	},
}

func upKeepCoverArtInDefaultTranscodings(ctx context.Context, tx *sql.Tx) error {
	for _, p := range coverArtPairs {
		if _, err := tx.ExecContext(ctx, `UPDATE transcoding SET command = ? WHERE command = ?`, p[1], p[0]); err != nil {
			return err
		}
	}
	return nil
}

func downKeepCoverArtInDefaultTranscodings(ctx context.Context, tx *sql.Tx) error {
	for _, p := range coverArtPairs {
		if _, err := tx.ExecContext(ctx, `UPDATE transcoding SET command = ? WHERE command = ?`, p[0], p[1]); err != nil {
			return err
		}
	}
	return nil
}

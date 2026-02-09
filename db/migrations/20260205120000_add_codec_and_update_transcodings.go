package migrations

import (
	"context"
	"database/sql"

	"github.com/navidrome/navidrome/model/id"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddCodecAndUpdateTranscodings, downAddCodecAndUpdateTranscodings)
}

func upAddCodecAndUpdateTranscodings(_ context.Context, tx *sql.Tx) error {
	// Add codec column to media_file.
	_, err := tx.Exec(`ALTER TABLE media_file ADD COLUMN codec VARCHAR(255) DEFAULT '' NOT NULL`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS media_file_codec ON media_file(codec)`)
	if err != nil {
		return err
	}

	// Update old AAC default (adts) to new default (ipod with fragmented MP4).
	// Only affects users who still have the unmodified old default command.
	_, err = tx.Exec(
		`UPDATE transcoding SET command = ? WHERE target_format = 'aac' AND command = ?`,
		"ffmpeg -i %s -ss %t -map 0:a:0 -b:a %bk -v 0 -c:a aac -f ipod -movflags frag_keyframe+empty_moov -",
		"ffmpeg -i %s -ss %t -map 0:a:0 -b:a %bk -v 0 -c:a aac -f adts -",
	)
	if err != nil {
		return err
	}

	// Add FLAC transcoding for existing installations that were seeded before FLAC was added.
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM transcoding WHERE target_format = 'flac'").Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		_, err = tx.Exec(
			"INSERT INTO transcoding (id, name, target_format, default_bit_rate, command) VALUES (?, ?, ?, ?, ?)",
			id.NewRandom(), "flac audio", "flac", 0,
			"ffmpeg -i %s -ss %t -map 0:a:0 -v 0 -c:a flac -f flac -",
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func downAddCodecAndUpdateTranscodings(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`DROP INDEX IF EXISTS media_file_codec`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`ALTER TABLE media_file DROP COLUMN codec`)
	return err
}

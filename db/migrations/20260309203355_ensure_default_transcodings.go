package migrations

import (
	"context"
	"database/sql"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model/id"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upEnsureDefaultTranscodings, downEnsureDefaultTranscodings)
}

func upEnsureDefaultTranscodings(_ context.Context, tx *sql.Tx) error {
	// Older installations may be missing default transcodings that were added
	// after the initial seeding (e.g., aac was added later than mp3/opus).
	// Insert any missing defaults without touching user-customized entries.
	for _, t := range consts.DefaultTranscodings {
		var count int
		err := tx.QueryRow("SELECT COUNT(*) FROM transcoding WHERE target_format = ?", t.TargetFormat).Scan(&count)
		if err != nil {
			return err
		}
		if count == 0 {
			_, err = tx.Exec(
				"INSERT INTO transcoding (id, name, target_format, default_bit_rate, command) VALUES (?, ?, ?, ?, ?)",
				id.NewRandom(), t.Name, t.TargetFormat, t.DefaultBitRate, t.Command,
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func downEnsureDefaultTranscodings(_ context.Context, tx *sql.Tx) error {
	return nil
}

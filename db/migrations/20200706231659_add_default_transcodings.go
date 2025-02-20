package migrations

import (
	"context"
	"database/sql"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model/id"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddDefaultTranscodings, downAddDefaultTranscodings)
}

func upAddDefaultTranscodings(_ context.Context, tx *sql.Tx) error {
	row := tx.QueryRow("SELECT COUNT(*) FROM transcoding")
	var count int
	err := row.Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	stmt, err := tx.Prepare("insert into transcoding (id, name, target_format, default_bit_rate, command) values (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}

	for _, t := range consts.DefaultTranscodings {
		_, err := stmt.Exec(id.NewRandom(), t.Name, t.TargetFormat, t.DefaultBitRate, t.Command)
		if err != nil {
			return err
		}
	}
	return nil
}

func downAddDefaultTranscodings(_ context.Context, tx *sql.Tx) error {
	return nil
}

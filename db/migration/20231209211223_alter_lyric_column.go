package migrations

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/navidrome/navidrome/model"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAlterLyricColumn, downAlterLyricColumn)
}

func upAlterLyricColumn(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `alter table media_file rename COLUMN lyrics TO lyrics_old`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `alter table media_file add lyrics JSONB default '[]';`)
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`update media_file SET lyrics = ? where id = ?`)
	if err != nil {
		return err
	}

	rows, err := tx.Query(`select id, lyrics_old FROM media_file WHERE lyrics_old <> '';`)
	if err != nil {
		return err
	}

	var id string
	var lyrics sql.NullString
	for rows.Next() {
		err = rows.Scan(&id, &lyrics)
		if err != nil {
			return err
		}

		if !lyrics.Valid {
			continue
		}

		lyrics, err := model.ToLyrics("xxx", lyrics.String)
		if err != nil {
			return err
		}

		text, err := json.Marshal(model.LyricList{*lyrics})
		if err != nil {
			return err
		}

		_, err = stmt.Exec(string(text), id)
		if err != nil {
			return err
		}
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `ALTER TABLE media_file DROP COLUMN lyrics_old;`)
	if err != nil {
		return err
	}

	notice(tx, "A full rescan should be performed to pick up additional lyrics (existing lyrics have been preserved)")
	return nil
}

func downAlterLyricColumn(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}

package migrations

import (
	"context"
	"database/sql"

	"github.com/navidrome/navidrome/conf"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upArtistLongerText, downArtistLongerText)
}

func upArtistLongerText(ctx context.Context, tx *sql.Tx) error {
	var err error
	switch conf.Server.DbDriver {
	case "sqlite3":
		return nil
	case "pgx":
		_, err = tx.Exec(`
ALTER TABLE artist ALTER COLUMN biography TYPE text USING biography::text;
ALTER TABLE artist ALTER COLUMN similar_artists TYPE text USING similar_artists::text;
`)
	}
	return err
}

func downArtistLongerText(ctx context.Context, tx *sql.Tx) error {
	var err error
	switch conf.Server.DbDriver {
	case "sqlite3":
		return nil
	case "pgx":
		_, err = tx.Exec(`
ALTER TABLE artist ALTER COLUMN biography TYPE varchar(255) USING biography::varchar(255);
ALTER TABLE artist ALTER COLUMN similar_artists TYPE varchar(255) USING similar_artists::varchar(255);
`)
	}
	return err
}

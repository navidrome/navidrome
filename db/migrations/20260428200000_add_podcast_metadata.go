package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddPodcastMetadata, downAddPodcastMetadata)
}

func upAddPodcastMetadata(ctx context.Context, tx *sql.Tx) error {
	sqls := []string{
		// podcast_channel — location, license, publisher
		`ALTER TABLE podcast_channel ADD COLUMN location_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE podcast_channel ADD COLUMN location_geo TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE podcast_channel ADD COLUMN location_osm TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE podcast_channel ADD COLUMN license TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE podcast_channel ADD COLUMN publisher_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE podcast_channel ADD COLUMN publisher_url TEXT NOT NULL DEFAULT ''`,
		// podcast_episode — location, license
		`ALTER TABLE podcast_episode ADD COLUMN location_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE podcast_episode ADD COLUMN location_geo TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE podcast_episode ADD COLUMN location_osm TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE podcast_episode ADD COLUMN license TEXT NOT NULL DEFAULT ''`,
		// channel_id/episode_id have no FK constraints: the put() helper serialises empty
		// strings as "" rather than NULL, which would violate a FK constraint. Cascade
		// delete is handled at the application layer instead.
		`CREATE TABLE podcast_funding (
			id         TEXT PRIMARY KEY,
			channel_id TEXT NOT NULL,
			url        TEXT NOT NULL DEFAULT '',
			text       TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL
		)`,
		`CREATE INDEX podcast_funding_channel_id ON podcast_funding(channel_id)`,
		`CREATE TABLE podcast_image (
			id         TEXT PRIMARY KEY,
			channel_id TEXT NOT NULL DEFAULT '',
			episode_id TEXT NOT NULL DEFAULT '',
			url        TEXT NOT NULL,
			width      INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL
		)`,
		`CREATE INDEX podcast_image_channel_id ON podcast_image(channel_id)`,
		`CREATE INDEX podcast_image_episode_id ON podcast_image(episode_id)`,
	}
	for _, s := range sqls {
		if _, err := tx.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

func downAddPodcastMetadata(ctx context.Context, tx *sql.Tx) error {
	return nil
}

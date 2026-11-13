package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddPodcastTier3, downAddPodcastTier3)
}

func upAddPodcastTier3(ctx context.Context, tx *sql.Tx) error {
	sqls := []string{
		// podcast:podping flag on channel
		`ALTER TABLE podcast_channel ADD COLUMN uses_podping INTEGER NOT NULL DEFAULT 0`,

		// podcast:podroll — recommended feeds listed by a channel
		`CREATE TABLE podcast_podroll (
            id          TEXT PRIMARY KEY,
            channel_id  TEXT NOT NULL,
            feed_guid   TEXT NOT NULL DEFAULT '',
            feed_url    TEXT NOT NULL DEFAULT '',
            title       TEXT NOT NULL DEFAULT '',
            sort_order  INTEGER NOT NULL DEFAULT 0,
            created_at  DATETIME NOT NULL
        )`,
		`CREATE INDEX podcast_podroll_channel_id ON podcast_podroll(channel_id)`,

		// podcast:liveItem — at most one active live item per channel
		`CREATE TABLE podcast_live_item (
            id                TEXT PRIMARY KEY,
            channel_id        TEXT NOT NULL,
            guid              TEXT NOT NULL DEFAULT '',
            title             TEXT NOT NULL DEFAULT '',
            status            TEXT NOT NULL DEFAULT 'pending',
            start_time        DATETIME,
            end_time          DATETIME,
            enclosure_url     TEXT NOT NULL DEFAULT '',
            enclosure_type    TEXT NOT NULL DEFAULT '',
            content_link_url  TEXT NOT NULL DEFAULT '',
            content_link_text TEXT NOT NULL DEFAULT '',
            created_at        DATETIME NOT NULL,
            updated_at        DATETIME NOT NULL
        )`,
		`CREATE UNIQUE INDEX podcast_live_item_channel_id ON podcast_live_item(channel_id)`,
	}
	for _, s := range sqls {
		if _, err := tx.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

func downAddPodcastTier3(ctx context.Context, tx *sql.Tx) error {
	return nil
}

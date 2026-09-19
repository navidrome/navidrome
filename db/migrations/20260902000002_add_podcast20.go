package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddPodcast20, downAddPodcast20)
}

func upAddPodcast20(ctx context.Context, tx *sql.Tx) error {
	sqls := []string{
		// podcast_channel — Tier 1
		`ALTER TABLE podcast_channel ADD COLUMN podcast_guid TEXT NOT NULL DEFAULT ''`,
		// podcast_channel — Tier 2
		`ALTER TABLE podcast_channel ADD COLUMN locked INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE podcast_channel ADD COLUMN locked_owner TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE podcast_channel ADD COLUMN medium TEXT NOT NULL DEFAULT 'podcast'`,
		`ALTER TABLE podcast_channel ADD COLUMN funding_url TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE podcast_channel ADD COLUMN funding_text TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE podcast_channel ADD COLUMN update_frequency TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE podcast_channel ADD COLUMN update_rrule TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE podcast_channel ADD COLUMN complete INTEGER NOT NULL DEFAULT 0`,
		// podcast_episode — Tier 1
		`ALTER TABLE podcast_episode ADD COLUMN season INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE podcast_episode ADD COLUMN season_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE podcast_episode ADD COLUMN episode_number TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE podcast_episode ADD COLUMN episode_display TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE podcast_episode ADD COLUMN chapters_url TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE podcast_episode ADD COLUMN chapters_type TEXT NOT NULL DEFAULT ''`,
		// podcast_episode — Tier 2
		`ALTER TABLE podcast_episode ADD COLUMN soundbite_start REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE podcast_episode ADD COLUMN soundbite_dur REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE podcast_episode ADD COLUMN soundbite_title TEXT NOT NULL DEFAULT ''`,
		// new tables
		`CREATE TABLE podcast_transcript (
			id         TEXT PRIMARY KEY,
			episode_id TEXT NOT NULL REFERENCES podcast_episode(id) ON DELETE CASCADE,
			url        TEXT NOT NULL,
			mime_type  TEXT NOT NULL DEFAULT '',
			language   TEXT NOT NULL DEFAULT '',
			rel        TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL
		)`,
		`CREATE INDEX podcast_transcript_episode_id ON podcast_transcript(episode_id)`,
		// channel_id/episode_id have no FK constraints: the put() helper serialises empty
		// strings as "" rather than NULL, which would violate a FK constraint. Cascade
		// delete is handled at the application layer instead.
		`CREATE TABLE podcast_person (
			id         TEXT PRIMARY KEY,
			channel_id TEXT,
			episode_id TEXT,
			name       TEXT NOT NULL,
			role       TEXT NOT NULL DEFAULT 'host',
			group_name TEXT NOT NULL DEFAULT 'cast',
			img        TEXT NOT NULL DEFAULT '',
			href       TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL
		)`,
		`CREATE INDEX podcast_person_channel_id ON podcast_person(channel_id)`,
		`CREATE INDEX podcast_person_episode_id ON podcast_person(episode_id)`,
	}
	for _, s := range sqls {
		if _, err := tx.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

func downAddPodcast20(ctx context.Context, tx *sql.Tx) error {
	return nil
}

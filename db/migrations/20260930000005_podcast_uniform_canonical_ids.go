package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upPodcastUniformCanonicalIds, downPodcastUniformCanonicalIds)
}

// podcastIDColumns lists every Navidrome-id-bearing podcast_* column, the same inventory
// uniform_canonical_ids (20260720015443) keeps for every other table - see this file's own
// upPodcastUniformCanonicalIds doc comment for why podcast ids need their own, later migration
// instead of just being added to that one's idColumns.
var podcastIDColumns = []struct{ table, col string }{
	{"podcast_channel", "id"},
	{"podcast_episode", "id"}, {"podcast_episode", "channel_id"}, {"podcast_episode", "stream_id"},
	{"podcast_transcript", "id"}, {"podcast_transcript", "episode_id"},
	{"podcast_person", "id"}, {"podcast_person", "channel_id"}, {"podcast_person", "episode_id"},
	{"podcast_podroll", "id"}, {"podcast_podroll", "channel_id"},
	{"podcast_live_item", "id"}, {"podcast_live_item", "channel_id"},
	{"podcast_funding", "id"}, {"podcast_funding", "channel_id"},
	{"podcast_image", "id"}, {"podcast_image", "channel_id"}, {"podcast_image", "episode_id"},
}

// upPodcastUniformCanonicalIds rewrites podcast_* ids to the same canonical 22-char base62
// encoding uniform_canonical_ids (20260720015443) already applied to every other table.
//
// It has to be a separate, later migration rather than an addition to that one's idColumns,
// for two independent reasons:
//  1. The podcast_* tables don't exist yet when 20260720015443 runs - the add_podcast* migrations
//     that create them are timestamped after it (2026-09-02, see add_podcast.go's own comment on
//     why) - so a SELECT against them there would fail outright, even on a fresh install.
//  2. Editing an already-applied migration's Go source has no runtime effect on any install that
//     already ran it: goose tracks migrations as applied-or-not by version, not by re-diffing
//     their source on every startup. An install that ran 20260720015443 before this feature
//     existed would never re-run it, no matter what idColumns says today.
func upPodcastUniformCanonicalIds(ctx context.Context, tx *sql.Tx) error {
	if err := buildIDMap(ctx, tx, podcastIDColumns); err != nil {
		return err
	}
	for _, tc := range podcastIDColumns {
		if err := applyIDMap(ctx, tx, tc.table, tc.col); err != nil {
			return fmt.Errorf("canonicalizing %s.%s: %w", tc.table, tc.col, err)
		}
	}
	_, err := tx.ExecContext(ctx, "DROP TABLE _id_map")
	return err
}

func downPodcastUniformCanonicalIds(ctx context.Context, tx *sql.Tx) error {
	return nil // irreversible data migration
}

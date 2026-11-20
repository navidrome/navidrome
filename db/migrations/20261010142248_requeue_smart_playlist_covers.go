package migrations

import (
	"context"
	"database/sql"
	"time"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upRequeueSmartPlaylistCovers, downRequeueSmartPlaylistCovers)
}

// Re-queue smart playlist covers that resolved while the playlist was still empty. Times are bound
// from Go: retry_at is compared as text against Go-formatted times, which CURRENT_TIMESTAMP doesn't match.
func upRequeueSmartPlaylistCovers(ctx context.Context, tx *sql.Tx) error {
	now := time.Now()
	_, err := tx.ExecContext(ctx, `
INSERT INTO artwork_queue (item_kind, item_id, image_type, priority, attempts, retry_at, enqueued_at)
SELECT 'pl', p.id, 'primary', 50, 0, ?, ?
FROM playlist p
LEFT JOIN item_artwork ia ON ia.item_kind = 'pl' AND ia.item_id = p.id AND ia.image_type = 'primary'
WHERE p.rules IS NOT NULL AND p.rules != '' AND p.song_count > 0 AND COALESCE(ia.hash, '') = ''
ON CONFLICT (item_kind, item_id, image_type) DO NOTHING`, now, now)
	return err
}

func downRequeueSmartPlaylistCovers(context.Context, *sql.Tx) error {
	return nil
}

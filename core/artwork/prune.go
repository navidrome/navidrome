package artwork

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// pruneMinAge guards the window between artwork insert and item_artwork upsert.
const pruneMinAge = time.Hour

func prune(ctx context.Context, ds model.DataStore, store *ImageStore) error {
	start := time.Now()
	defer func() { log.Debug(ctx, "Artwork: Prune finished", "elapsed", time.Since(start)) }()
	repo := ds.Artwork(ctx)

	purged, err := repo.PurgeDanglingItemArtwork()
	if err != nil {
		return err
	}
	if purged > 0 {
		log.Info(ctx, "Artwork: Purged dangling item state", "count", purged)
	}

	// Queue rows for deleted entities would otherwise retry forever (Get -> not found -> failed).
	queuePurged, err := ds.ArtworkQueue(ctx).PurgeDangling()
	if err != nil {
		return err
	}
	if queuePurged > 0 {
		log.Info(ctx, "Artwork: Purged dangling queue rows", "count", queuePurged)
	}

	// Files younger than the grace window may belong to acquisitions whose rows aren't committed yet.
	cutoff := time.Now().Add(-pruneMinAge)
	orphans, err := repo.DeleteOrphans(cutoff)
	if err != nil {
		return err
	}
	if orphans > 0 {
		log.Info(ctx, "Artwork: Removed orphan images", "count", orphans)
	}

	// Read after the delete, so the sweep below reclaims the files of the rows just removed.
	mimes, err := repo.GetAllMimes()
	if err != nil {
		return err
	}
	removed, err := store.Sweep(ctx, cutoff, func(hash, ext string) bool {
		// A known hash under a stale extension is a superseded mime variant — reclaim it.
		m, ok := mimes[hash]
		return ok && ext == extForMime(m)
	})
	if err != nil {
		return err
	}
	if removed > 0 {
		log.Info(ctx, "Artwork: Swept stray files", "count", removed)
	}
	return nil
}

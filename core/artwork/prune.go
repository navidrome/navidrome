package artwork

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// pruneMinAge guards the window between artwork insert and item_artwork upsert.
const pruneMinAge = time.Hour

func Prune(ctx context.Context, ds model.DataStore, store *ImageStore) error {
	repo := ds.Artwork(ctx)
	// One grace cutoff for both the DB orphan check and the file sweep: files younger
	// than the window may belong to acquisitions whose rows aren't committed yet.
	cutoff := time.Now().Add(-pruneMinAge)
	candidates, err := repo.GetOrphanHashes(cutoff)
	if err != nil {
		return err
	}
	if len(candidates) > 0 {
		arts, err := repo.GetImages(candidates)
		if err != nil {
			return err
		}
		if err := repo.DeleteOrphans(cutoff, candidates); err != nil {
			return err
		}
		// DeleteOrphans may spare candidates reacquired since the snapshot; only remove files
		// for rows actually gone (absent from the post-delete re-read).
		survivors, err := repo.GetImages(candidates)
		if err != nil {
			return err
		}
		removed := 0
		for _, h := range candidates {
			if _, ok := survivors[h]; ok {
				continue
			}
			// A spared fresh file is at worst a stray a later sweep reclaims;
			// Worker.RunPrune serializes prune against in-flight acquisitions.
			if err := store.Remove(h, arts[h].Mime, cutoff); err != nil {
				log.Warn(ctx, "Prune: could not remove artwork file", "hash", h, err)
			}
			removed++
		}
		log.Info(ctx, "Prune: removed orphan artwork", "count", removed)
	}

	mimes, err := repo.GetAllMimes()
	if err != nil {
		return err
	}
	removed, err := store.Sweep(cutoff, func(hash, ext string) bool {
		// A known hash under a stale extension is a superseded mime variant — reclaim it.
		m, ok := mimes[hash]
		return ok && ext == extForMime(m)
	})
	if err != nil {
		return err
	}
	if removed > 0 {
		log.Info(ctx, "Prune: swept stray artwork files", "count", removed)
	}
	return nil
}

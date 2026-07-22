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
	orphans, err := repo.GetOrphanHashes(cutoff)
	if err != nil {
		return err
	}
	if len(orphans) > 0 {
		arts, err := repo.GetImages(orphans)
		if err != nil {
			return err
		}
		if err := repo.DeleteImages(orphans...); err != nil {
			return err
		}
		for _, a := range arts {
			if err := store.Remove(a.Hash, a.Mime); err != nil {
				log.Warn(ctx, "Prune: could not remove artwork file", "hash", a.Hash, err)
			}
		}
		log.Info(ctx, "Prune: removed orphan artwork", "count", len(orphans))
	}

	hashes, err := repo.GetAllHashes()
	if err != nil {
		return err
	}
	known := make(map[string]struct{}, len(hashes))
	for _, h := range hashes {
		known[h] = struct{}{}
	}
	removed, err := store.Sweep(cutoff, func(hash string) bool {
		_, ok := known[hash]
		return ok
	})
	if err != nil {
		return err
	}
	if removed > 0 {
		log.Info(ctx, "Prune: swept stray artwork files", "count", removed)
	}
	return nil
}

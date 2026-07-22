package artwork

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/core/artwork/originals"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// pruneMinAge guards the window between artwork insert and item_artwork upsert.
const pruneMinAge = time.Hour

func Prune(ctx context.Context, ds model.DataStore, store *originals.Store) error {
	repo := ds.Artwork(ctx)
	orphans, err := repo.GetOrphanHashes(time.Now().Add(-pruneMinAge))
	if err != nil {
		return err
	}
	if len(orphans) > 0 {
		arts, err := repo.GetBatch(orphans)
		if err != nil {
			return err
		}
		if err := repo.Delete(orphans...); err != nil {
			return err
		}
		for _, a := range arts {
			if err := store.Remove(a.Hash, a.Mime); err != nil {
				log.Warn(ctx, "Prune: could not remove artwork file", "hash", a.Hash, err)
			}
		}
		log.Info(ctx, "Prune: removed orphan artwork", "count", len(orphans))
	}

	removed, err := store.Sweep(func(hash string) bool {
		_, err := repo.Get(hash)
		return !errors.Is(err, model.ErrNotFound)
	})
	if err != nil {
		return err
	}
	if removed > 0 {
		log.Info(ctx, "Prune: swept stray artwork files", "count", removed)
	}
	return nil
}

package artwork

import (
	"context"
	"fmt"

	"github.com/navidrome/navidrome/model"
)

// Refresh clears an item's resolved artwork state and re-queues it at Bump priority, so a
// deliberate refresh (image upload or manual re-resolve) drops the current pick and re-resolves it.
func Refresh(ctx context.Context, ds model.DataStore, kind, id string) error {
	if err := ds.Artwork(ctx).DeleteForItem(kind, id); err != nil {
		return fmt.Errorf("clearing artwork state: %w", err)
	}
	item := model.ArtworkQueueItem{ItemKind: kind, ItemID: id, ImageType: model.ImageTypePrimary, Priority: model.ArtworkPriorityBump}
	if err := ds.ArtworkQueue(ctx).Enqueue(item); err != nil {
		return fmt.Errorf("enqueuing artwork refresh: %w", err)
	}
	return nil
}

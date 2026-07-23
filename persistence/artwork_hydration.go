package persistence

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

// hydrateItemImages returns per-item artwork info for a fetched page via one batched query per kind
// (never a join, see spec §6). On error it logs and returns an empty map so the page still renders.
func hydrateItemImages(ctx context.Context, db dbx.Builder, kind string, ids []string) map[string]model.ItemArtworkInfo {
	if len(ids) == 0 {
		return map[string]model.ItemArtworkInfo{}
	}
	infos, err := NewArtworkRepository(ctx, db).GetInfoForItems(kind, ids)
	if err != nil {
		log.Error(ctx, "Failed to hydrate artwork info onto page", "kind", kind, err)
		return map[string]model.ItemArtworkInfo{}
	}
	return infos
}

// applyItemImage copies a hydration entry onto img; a missing entry leaves it zero (unresolved).
func applyItemImage(infos map[string]model.ItemArtworkInfo, id string, img *model.ItemImage) {
	if info, ok := infos[id]; ok {
		img.ImageHash = info.Hash
		img.ImageAbsent = info.Absent()
	}
}

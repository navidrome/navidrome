package persistence

import (
	"context"
	"iter"
	"slices"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

// artworkChunkSize bounds the id IN-list of each chunk fetch, keeping it under SQLite's bound
// parameter limit.
const artworkChunkSize = 500

// streamByIDs yields the rows of ids in chunks, fetching each chunk through the caller's hydrating
// fetch. Resolving ids first keeps OFFSET out of the joined query (spec §6).
func streamByIDs[S ~[]T, T any](ids []string, fetch func(chunk []string) (S, error)) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for chunk := range slices.Chunk(ids, artworkChunkSize) {
			rows, err := fetch(chunk)
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}
			for _, row := range rows {
				if !yield(row, nil) {
					return
				}
			}
		}
	}
}

// chunkOptions narrows the caller's options to a chunk of ids, dropping Max/Offset (the id pre-pass
// already consumed them) and reusing its Sort, which may be a seeded random expression by now.
func chunkOptions(options []model.QueryOptions, idField string) func([]string) model.QueryOptions {
	var base model.QueryOptions
	if len(options) > 0 {
		base = model.QueryOptions{Sort: options[0].Sort, Order: options[0].Order, Filters: options[0].Filters}
	}
	return func(chunk []string) model.QueryOptions {
		opts := base
		opts.Filters = Eq{idField: chunk}
		if base.Filters != nil {
			opts.Filters = And{base.Filters, opts.Filters}
		}
		return opts
	}
}

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
		img.BlurHash = info.BlurHash
	}
}

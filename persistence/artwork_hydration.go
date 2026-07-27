package persistence

import (
	"context"
	"iter"
	"slices"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

// Keeps the id IN-list under SQLite's bound-parameter limit; a whole multiple of artworkBatchSize
// so a page re-chunks into even hydration batches.
const artworkChunkSize = artworkBatchSize * 3

// streamByIDs yields rows in id chunks through the caller's hydrating fetch. Resolving ids first
// keeps OFFSET out of the joined query.
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

// chunkOptions narrows options to a chunk of ids, dropping Max/Offset (the id pre-pass already
// consumed them) and reusing Sort, which may be a seeded random expression by now.
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

// hydrateItemImages returns per-item artwork info in one batched query per kind. On error it logs
// and returns an empty map, so the page still renders.
func hydrateItemImages(ctx context.Context, db dbx.Builder, kind model.Kind, ids []string) map[string]model.ItemArtworkInfo {
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
		*img = info.Image()
	}
}

// hydrateMediaFileArtwork mirrors MediaFile.CoverArtID: an embedded-eligible file with resolved own
// art uses it, else it falls back to the album's.
func hydrateMediaFileArtwork(ctx context.Context, db dbx.Builder, mfs model.MediaFiles) {
	if len(mfs) == 0 {
		return
	}
	albumIDs := make([]string, len(mfs))
	var eligibleIDs []string
	for i := range mfs {
		albumIDs[i] = mfs[i].AlbumID
		if mfs[i].HasCoverArt && conf.Server.EnableMediaFileCoverArt {
			eligibleIDs = append(eligibleIDs, mfs[i].ID)
		}
	}
	albumInfos := hydrateItemImages(ctx, db, model.KindAlbumArtwork, albumIDs)
	mfInfos := hydrateItemImages(ctx, db, model.KindMediaFileArtwork, eligibleIDs)
	for i := range mfs {
		mf := &mfs[i]
		applyItemImage(albumInfos, mf.AlbumID, &mf.AlbumImage)
		eligible := mf.HasCoverArt && conf.Server.EnableMediaFileCoverArt
		ownInfo, ownResolved := mfInfos[mf.ID]
		if eligible && ownResolved && !ownInfo.Absent() {
			mf.ItemImage = ownInfo.Image()
			continue
		}
		ownWontResolve := !eligible || (ownResolved && ownInfo.Absent())
		// Inherit the album hash only when serving returns those exact bytes: a multi-disc track is
		// served disc art, and an eligible-but-unresolved one still extracts its own embedded image.
		if album, ok := albumInfos[mf.AlbumID]; ok && !album.Absent() {
			if mf.DiscNumber == 0 && ownWontResolve {
				mf.ItemImage = album.Image()
			}
			continue
		}
		// Mark absent only when serving would definitively yield a placeholder; disc art and
		// still-extractable embedded art both keep a track requestable.
		if mf.DiscNumber > 0 {
			continue
		}
		if album, ok := albumInfos[mf.AlbumID]; ok && album.Absent() && ownWontResolve {
			mf.ImageAbsent = true
		}
	}
}

// hydrateCursor hydrates a streamed cursor in batches, avoiding a per-row query.
func hydrateCursor[T any](cursor iter.Seq2[T, error], hydrate func([]T)) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		buf := make([]T, 0, artworkBatchSize)
		flush := func() bool {
			hydrate(buf)
			for i := range buf {
				if !yield(buf[i], nil) {
					return false
				}
			}
			buf = buf[:0]
			return true
		}
		for row, err := range cursor {
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}
			buf = append(buf, row)
			if len(buf) == artworkBatchSize && !flush() {
				return
			}
		}
		if len(buf) > 0 {
			flush()
		}
	}
}

// hydratePlaylistTrackArtwork hydrates the MediaFile embedded in each playlist track.
func hydratePlaylistTrackArtwork(ctx context.Context, db dbx.Builder, tracks model.PlaylistTracks) {
	if len(tracks) == 0 {
		return
	}
	mfs := make(model.MediaFiles, len(tracks))
	for i := range tracks {
		mfs[i] = tracks[i].MediaFile
	}
	hydrateMediaFileArtwork(ctx, db, mfs)
	for i := range tracks {
		tracks[i].MediaFile = mfs[i]
	}
}

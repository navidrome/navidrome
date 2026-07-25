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

// artworkChunkSize bounds each chunk fetch's id IN-list (under SQLite's bound-parameter limit); a
// whole multiple of artworkBatchSize so a page re-chunks into even hydration batches.
const artworkChunkSize = artworkBatchSize * 3

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
		img.ImageHash = info.Hash
		img.ImageAbsent = info.Absent()
		img.BlurHash = info.BlurHash
	}
}

// hydrateMediaFileArtwork mirrors MediaFile.CoverArtID: an embedded-eligible file with resolved own
// art uses it, else it falls back to the album's. Two batched item_artwork lookups, never a join.
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
			mf.ImageHash = ownInfo.Hash // own resolved art wins
			mf.BlurHash = ownInfo.BlurHash
			continue
		}
		// Fallback (see MediaFile.CoverArtID): inherit a found album hash for optimistic caching,
		// but only for a single-disc track. A multi-disc track emits a dc- id served from
		// disc-specific art of unknown identity, so stamping the album hash would advertise a
		// wrong content-version; leave it bare (the served response still carries a correct ETag).
		if album, ok := albumInfos[mf.AlbumID]; ok && !album.Absent() {
			if mf.DiscNumber == 0 {
				mf.ImageHash = album.Hash
				mf.BlurHash = album.BlurHash
			}
			continue
		}
		// Nothing found. Mark absent only when serving would definitively yield a placeholder:
		// a single-disc track whose album is known-absent and whose own art won't resolve. A
		// multi-disc track resolves disc art provisionally (never known-absent), and an
		// eligible-but-unresolved track can still extract its own embedded art — both stay
		// requestable.
		if mf.DiscNumber > 0 {
			continue
		}
		ownWontResolve := !eligible || (ownResolved && ownInfo.Absent())
		if album, ok := albumInfos[mf.AlbumID]; ok && album.Absent() && ownWontResolve {
			mf.ImageAbsent = true
		}
	}
}

// hydrateCursor buffers a streamed page into batches and hydrates each before yielding, so a
// cursor carries the same artwork state as a fetched page without a per-row query.
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

// hydratePlaylistTrackArtwork hydrates the MediaFile embedded in each playlist track, so a track
// reached through a playlist carries the same artwork state as one reached through the songs list.
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

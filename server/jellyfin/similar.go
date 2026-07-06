package jellyfin

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/utils/req"
	"github.com/navidrome/navidrome/utils/slice"
)

// similarQuickWait bounds how long a Similar request waits on the external metadata provider
// (Last.fm etc.). A cached artist resolves well within it and returns immediately; a cold lookup
// exceeds it, so we return an empty result now and let the lookup finish caching in the background,
// making a later load instant. Without this, clients (Jellify request Similar for many items at
// once) stall the whole screen on a synchronous Last.fm round-trip per item.
const similarQuickWait = 500 * time.Millisecond

// awaitSimilar runs fetch on a background context — so a cold external lookup completes and caches
// even after this request returns — and returns its result if ready within similarQuickWait,
// otherwise an empty result. The background context carries the caller's user (external info is
// user-agnostic, but repositories still expect a user in context) but not the request's
// cancellation, so navigating away doesn't abort the cache warm-up.
func awaitSimilar(ctx context.Context, fetch func(context.Context) dto.QueryResult) dto.QueryResult {
	u, _ := request.UserFrom(ctx)
	bgCtx := request.WithUser(context.Background(), u)
	resCh := make(chan dto.QueryResult, 1)
	go func() { resCh <- fetch(bgCtx) }()
	select {
	case res := <-resCh:
		return res
	case <-time.After(similarQuickWait):
		return result(nil, 0, 0)
	}
}

// getSimilarArtists answers GET /Artists/{itemId}/Similar with artists related to the given artist,
// sourced from the same external.Provider (Last.fm etc.) that powers Subsonic's getArtistInfo2.
// Only artists present in the library are returned, so each one is navigable. Any provider error
// (agent disabled, artist unknown, nothing found) degrades to an empty result rather than the 404
// the client would otherwise keep retrying.
func (api *Router) getSimilarArtists(w http.ResponseWriter, r *http.Request) {
	id := dto.DecodeID(chi.URLParam(r, "itemId"))
	limit := req.Params(r).IntOr("limit", 20)
	api.ok(w, r, awaitSimilar(r.Context(), func(ctx context.Context) dto.QueryResult {
		return api.similarArtists(ctx, id, limit)
	}))
}

// getSimilarItems answers GET /Items/{itemId}/Similar with items of the same kind as the target:
// similar songs for a track, similar albums for an album, similar artists for an artist. Jellify
// uses this for non-artist items; /Artists/{id}/Similar covers artists directly. An unresolvable id
// yields an empty result (not 404) so the client stops retrying.
func (api *Router) getSimilarItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := dto.DecodeID(chi.URLParam(r, "itemId"))
	limit := req.Params(r).IntOr("limit", 20)

	entity, err := model.GetEntityByID(ctx, api.ds, id)
	if err != nil {
		api.ok(w, r, result(nil, 0, 0))
		return
	}
	api.ok(w, r, awaitSimilar(ctx, func(ctx context.Context) dto.QueryResult {
		switch entity.(type) {
		case *model.Artist:
			return api.similarArtists(ctx, id, limit)
		case *model.Album:
			return api.similarAlbums(ctx, id, limit)
		default: // *model.MediaFile
			return api.similarSongs(ctx, id, limit)
		}
	}))
}

func (api *Router) similarArtists(ctx context.Context, id string, limit int) dto.QueryResult {
	artist, err := api.provider.UpdateArtistInfo(ctx, id, limit, false)
	if err != nil {
		log.Debug(ctx, "Jellyfin API: no similar artists", "id", id, err)
		return result(nil, 0, 0)
	}
	present := slice.Filter(artist.SimilarArtists, func(a model.Artist) bool { return a.ID != "" })
	items := slice.Map(present, dto.ArtistToBaseItem)
	return result(items, len(items), 0)
}

func (api *Router) similarSongs(ctx context.Context, id string, limit int) dto.QueryResult {
	songs, err := api.provider.SimilarSongs(ctx, id, limit)
	if err != nil {
		log.Debug(ctx, "Jellyfin API: no similar songs", "id", id, err)
		return result(nil, 0, 0)
	}
	items := slice.Map(songs, dto.SongToBaseItem)
	return result(items, len(items), 0)
}

// similarAlbums derives similar albums from the provider's similar-songs signal (Navidrome has no
// direct "similar albums" source), keeping each album once in first-seen order and resolving it to
// a full model.Album so the mapped item carries cover art and metadata.
func (api *Router) similarAlbums(ctx context.Context, id string, limit int) dto.QueryResult {
	songs, err := api.provider.SimilarSongs(ctx, id, limit*5)
	if err != nil {
		log.Debug(ctx, "Jellyfin API: no similar albums", "id", id, err)
		return result(nil, 0, 0)
	}
	seen := make(map[string]bool, limit)
	var items []dto.BaseItemDto
	for _, s := range songs {
		if s.AlbumID == "" || seen[s.AlbumID] {
			continue
		}
		seen[s.AlbumID] = true
		if al, err := api.ds.Album(ctx).Get(s.AlbumID); err == nil {
			items = append(items, dto.AlbumToBaseItem(*al))
			if len(items) >= limit {
				break
			}
		}
	}
	return result(items, len(items), 0)
}

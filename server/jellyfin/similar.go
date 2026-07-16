package jellyfin

import (
	"context"
	"fmt"
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

// similarWait bounds how long a Similar request waits for the provider fetch. Returning the real
// result beats an instant empty list, which clients cache as "no similar items exist". A var so
// tests can shorten it.
var similarWait = 10 * time.Second

const (
	defaultSimilarLimit = 20
	maxSimilarLimit     = 100
	// A mix is a playback queue, not a "related items" list: Finamp's Radio Mix asks for 250, so the
	// Similar ceiling would truncate it. Real Jellyfin builds mixes from a 200-track genre query.
	maxInstantMixLimit = 500
)

// similarFetchTimeout bounds the detached background fetch so a hung provider can't hold a goroutine
// indefinitely.
const similarFetchTimeout = time.Minute

// awaitSimilar runs fetch on a detached background context (so it completes and caches even if the
// request times out or the client disconnects), waiting up to similarWait then answering empty.
// Identical concurrent requests share one fetch via singleflight; the key includes the user since
// mapped items embed that user's annotations.
func (api *Router) awaitSimilar(ctx context.Context, id string, limit int, fetch func(context.Context) dto.QueryResult) dto.QueryResult {
	u, _ := request.UserFrom(ctx)
	key := fmt.Sprintf("%s|%s|%d", u.ID, id, limit)
	ch := api.similarFlight.DoChan(key, func() (any, error) {
		bgCtx, cancel := context.WithTimeout(request.WithUser(context.Background(), u), similarFetchTimeout)
		defer cancel()
		return fetch(bgCtx), nil
	})
	select {
	case res := <-ch:
		return res.Val.(dto.QueryResult)
	case <-time.After(similarWait):
		return result(nil, 0, 0)
	}
}

// getSimilarArtists answers GET /Artists/{itemId}/Similar with related artists from the same
// external.Provider that powers Subsonic's getArtistInfo2. Only artists present in the library are
// returned. Any provider error degrades to an empty result, not a 404 the client would keep retrying.
func (api *Router) getSimilarArtists(w http.ResponseWriter, r *http.Request) {
	id := api.resolveItemID(r.Context(), dto.DecodeID(chi.URLParam(r, "itemId")))
	limit := clampLimit(req.Params(r).IntOr("limit", 0), defaultSimilarLimit, maxSimilarLimit)
	api.ok(w, r, api.awaitSimilar(r.Context(), id, limit, func(ctx context.Context) dto.QueryResult {
		return api.similarArtists(ctx, id, limit)
	}))
}

// getSimilarItems answers GET /Items/{itemId}/Similar with items of the target's kind: similar
// songs for a track, albums for an album, artists for an artist. An unresolvable id yields an empty
// result (not 404) so the client stops retrying.
func (api *Router) getSimilarItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := api.resolveItemID(ctx, dto.DecodeID(chi.URLParam(r, "itemId")))
	limit := clampLimit(req.Params(r).IntOr("limit", 0), defaultSimilarLimit, maxSimilarLimit)

	entity, err := model.GetEntityByID(ctx, api.ds, id)
	if err != nil {
		api.ok(w, r, result(nil, 0, 0))
		return
	}
	api.ok(w, r, api.awaitSimilar(ctx, id, limit, func(ctx context.Context) dto.QueryResult {
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

// getInstantMix answers GET /Items/{itemId}/InstantMix. Finamp plays exactly what is returned, so
// a track seed leads its own mix; provider errors and unknown seeds degrade to seed-only/empty
// results, never a 404 the client would surface as an error.
func (api *Router) getInstantMix(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := api.resolveItemID(ctx, dto.DecodeID(chi.URLParam(r, "itemId")))
	limit := clampLimit(req.Params(r).IntOr("limit", 0), defaultSimilarLimit, maxInstantMixLimit)

	entity, err := model.GetEntityByID(ctx, api.ds, id)
	if err != nil {
		api.ok(w, r, result(nil, 0, 0))
		return
	}
	mf, isSong := entity.(*model.MediaFile)
	if isSong {
		if u, _ := request.UserFrom(ctx); !u.HasLibraryAccess(mf.LibraryID) {
			api.ok(w, r, result(nil, 0, 0))
			return
		}
	}
	// Prefixed key: a mix must not share the singleflight/cache slot with a Similar request.
	tail := api.awaitSimilar(ctx, "mix|"+id, limit, func(ctx context.Context) dto.QueryResult {
		return api.similarSongs(ctx, id, limit)
	})
	if !isSong {
		// Container seeds: the provider's similar songs already blend the seed's own tracks.
		api.ok(w, r, tail)
		return
	}
	// The seed leads the mix and must not depend on the provider: a slow or failing provider times
	// the await out with an empty tail, but the tapped track still plays.
	items := []dto.BaseItemDto{dto.SongToBaseItem(*mf, nil)}
	for _, it := range tail.Items {
		if len(items) >= limit {
			break
		}
		if it.Id != items[0].Id {
			items = append(items, it)
		}
	}
	api.ok(w, r, result(items, len(items), 0))
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
	// Filter to the caller's libraries; the provider can return songs from any library.
	u, _ := request.UserFrom(ctx)
	var items []dto.BaseItemDto
	for _, mf := range songs {
		if u.HasLibraryAccess(mf.LibraryID) {
			items = append(items, dto.SongToBaseItem(mf, nil))
		}
	}
	return result(items, len(items), 0)
}

// similarAlbums derives similar albums from the provider's similar-songs signal (there's no direct
// "similar albums" source), keeping each album once in first-seen order and resolving it to a full
// model.Album for cover art and metadata.
func (api *Router) similarAlbums(ctx context.Context, id string, limit int) dto.QueryResult {
	songs, err := api.provider.SimilarSongs(ctx, id, limit*5)
	if err != nil {
		log.Debug(ctx, "Jellyfin API: no similar albums", "id", id, err)
		return result(nil, 0, 0)
	}
	u, _ := request.UserFrom(ctx)
	seen := make(map[string]bool, limit)
	var items []dto.BaseItemDto
	for _, s := range songs {
		if s.AlbumID == "" || seen[s.AlbumID] {
			continue
		}
		seen[s.AlbumID] = true
		if al, err := api.ds.Album(ctx).Get(s.AlbumID); err == nil && u.HasLibraryAccess(al.LibraryID) {
			items = append(items, dto.AlbumToBaseItem(*al))
			if len(items) >= limit {
				break
			}
		}
	}
	return result(items, len(items), 0)
}

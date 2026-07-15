package jellyfin

import (
	"net/http"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/utils/req"
)

// getArtists handles GET /Artists (performing artists, Finamp's "Artists" tab); getAlbumArtists
// handles GET /Artists/AlbumArtists (album artists only). Distinct roles, so composers/arrangers
// don't appear identically in both.
func (api *Router) getArtists(w http.ResponseWriter, r *http.Request) {
	api.listArtistsByRole(w, r, model.RoleArtist)
}

func (api *Router) getAlbumArtists(w http.ResponseWriter, r *http.Request) {
	api.listArtistsByRole(w, r, model.RoleAlbumArtist)
}

// listArtistsByRole is the shared body of the /Artists* handlers, scoping to ParentId's library
// when accessible (like queryItems) or all accessible libraries otherwise.
func (api *Router) listArtistsByRole(w http.ResponseWriter, r *http.Request, role model.Role) {
	ctx := r.Context()
	p := req.Params(r)
	opts := model.QueryOptions{Offset: p.IntOr("startindex", 0), Max: p.IntOr("limit", 0)}
	applySort(&opts, "MusicArtist", p.StringOr("sortby", ""), p.StringOr("sortorder", ""))

	scopeIDs, _ := resolveLibraryScope(ctx, dto.DecodeID(p.StringOr("parentid", "")))
	// Only the fields listArtists reads; /Artists has no favorites filter, so favOnly stays false.
	// Finamp's artist tab sends GenreIds when a genre filter is active.
	q := itemsQuery{
		scopeIDs: scopeIDs,
		genreIds: decodedQueryIDs(r, "genreids"),
		search:   searchTerm(p),
	}
	if q.search != "" {
		opts.Max = clampLimit(opts.Max, defaultSearchLimit, maxSearchLimit)
	}

	res, err := api.listArtists(ctx, opts, q, role)
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	api.ok(w, r, res)
}

// getGenres handles /Genres and /MusicGenres. Genres are global, so no library scoping applies.
func (api *Router) getGenres(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p := req.Params(r)
	opts := model.QueryOptions{Offset: p.IntOr("startindex", 0), Max: p.IntOr("limit", 0)}
	res, err := api.listGenres(ctx, opts)
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	api.ok(w, r, res)
}

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
	// Finamp's artist tab sends GenreIds when a genre filter is active.
	genreIds := decodedQueryIDs(r, "genreids")

	res, err := api.listArtists(ctx, opts, genreIds, scopeIDs, p.StringOr("searchterm", ""), false, role)
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

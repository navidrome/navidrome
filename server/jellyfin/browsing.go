package jellyfin

import (
	"net/http"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/utils/req"
)

// getArtists handles GET /Artists — performing artists (Finamp's "Artists" tab). getAlbumArtists
// handles GET /Artists/AlbumArtists — album artists only ("Album Artists" tab). They're distinct
// roles: without this, both listed every participant (composers, arrangers, ...) identically.
func (api *Router) getArtists(w http.ResponseWriter, r *http.Request) {
	api.listArtistsByRole(w, r, model.RoleArtist)
}

func (api *Router) getAlbumArtists(w http.ResponseWriter, r *http.Request) {
	api.listArtistsByRole(w, r, model.RoleAlbumArtist)
}

// listArtistsByRole is the shared body of the /Artists* handlers. It defaults to the user's
// accessible libraries but narrows to a single one when ParentId names a library the user can
// access, matching how queryItems treats ParentId as a UserView id.
func (api *Router) listArtistsByRole(w http.ResponseWriter, r *http.Request, role model.Role) {
	ctx := r.Context()
	p := req.Params(r)
	opts := model.QueryOptions{Offset: p.IntOr("startindex", 0), Max: p.IntOr("limit", 0)}
	applySort(&opts, "MusicArtist", p.StringOr("sortby", ""), p.StringOr("sortorder", ""))

	scopeIDs, _ := resolveLibraryScope(ctx, dto.DecodeID(p.StringOr("parentid", "")))

	res, err := api.listArtists(ctx, opts, scopeIDs, p.StringOr("searchterm", ""), false, role)
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	api.ok(w, r, res)
}

// getGenres handles /Genres and /MusicGenres. Genres are global (see listGenres), so no
// library scoping is applied here.
func (api *Router) getGenres(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p := req.Params(r)
	opts := model.QueryOptions{Offset: p.IntOr("StartIndex", 0), Max: p.IntOr("Limit", 0)}
	res, err := api.listGenres(ctx, opts)
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	api.ok(w, r, res)
}

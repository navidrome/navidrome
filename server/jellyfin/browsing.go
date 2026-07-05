package jellyfin

import (
	"net/http"
	"strconv"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/req"
)

// getArtists handles /Artists and /Artists/AlbumArtists. It defaults to the user's
// accessible libraries but narrows to a single one when ParentId names a library the
// user can access, matching how queryItems treats ParentId as a UserView id.
func (api *Router) getArtists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p := req.Params(r)
	opts := model.QueryOptions{Offset: p.IntOr("StartIndex", 0), Max: p.IntOr("Limit", 0)}
	applySort(&opts, "MusicArtist", p.StringOr("SortBy", ""), p.StringOr("SortOrder", ""))

	scopeIDs := accessibleLibraryIDs(ctx)
	if parentId := p.StringOr("ParentId", ""); parentId != "" {
		if libID, err := strconv.Atoi(parentId); err == nil {
			if u, _ := request.UserFrom(ctx); u.HasLibraryAccess(libID) {
				scopeIDs = []int{libID}
			}
		}
	}

	res, err := api.listArtists(ctx, opts, scopeIDs, p.StringOr("SearchTerm", ""), false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	api.ok(w, r, res)
}

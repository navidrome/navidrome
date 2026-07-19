package jellyfin

import (
	"net/http"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/utils/req"
	"github.com/navidrome/navidrome/utils/slice"
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

// getStudios handles GET /Studios, exposing record labels (Jellyfin's audio "studio" source) as
// Studio items. Global, like genres.
func (api *Router) getStudios(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p := req.Params(r)
	labels, err := api.ds.Tag(ctx).GetAll(model.TagRecordLabel, model.QueryOptions{Sort: "tag_value"})
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	items := slice.Map(labels, dto.StudioToBaseItem)
	offset, max := p.IntOr("startindex", 0), p.IntOr("limit", 0)
	api.ok(w, r, result(paginate(items, offset, max), len(items), offset))
}

// getQueryFiltersLegacy handles GET /Items/Filters. Genres reuse the global genre list; Years are
// distinct album years. Tags/OfficialRatings have no music source, so they are always empty.
func (api *Router) getQueryFiltersLegacy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	genres, err := api.ds.Genre(ctx).GetAll(model.QueryOptions{Sort: "name"})
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	years, err := api.ds.Album(ctx).GetYears()
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	api.ok(w, r, dto.QueryFiltersLegacy{
		Genres:          slice.Map(genres, func(g model.Genre) string { return g.Name }),
		Tags:            []string{},
		OfficialRatings: []string{},
		Years:           years,
	})
}

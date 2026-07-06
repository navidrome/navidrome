package jellyfin

import (
	"net/http"

	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
)

// getUserViews returns one CollectionFolder view per accessible library, so clients browse each
// library as its own top-level view rather than one aggregate.
func (api *Router) getUserViews(w http.ResponseWriter, r *http.Request) {
	u, _ := request.UserFrom(r.Context())
	views := make([]dto.BaseItemDto, 0, len(u.Libraries))
	for _, lib := range u.Libraries {
		views = append(views, libraryView(lib))
	}
	api.ok(w, r, dto.QueryResult{Items: views, TotalRecordCount: len(views), StartIndex: 0})
}

func (api *Router) getCurrentUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	u, _ := request.UserFrom(ctx)
	api.ok(w, r, userToDto(&u, api.serverName(), api.serverID(ctx)))
}

func (api *Router) getPublicUsers(w http.ResponseWriter, r *http.Request) {
	api.ok(w, r, []dto.UserDto{}) // manual login only
}

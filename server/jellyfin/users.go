package jellyfin

import (
	"net/http"
	"strconv"

	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
)

// getUserViews returns one CollectionFolder view per library the current user can access,
// so Jellyfin clients browse each library as its own top-level view (instead of a single
// aggregate "music" view spanning every library).
func (api *Router) getUserViews(w http.ResponseWriter, r *http.Request) {
	u, _ := request.UserFrom(r.Context())
	views := make([]dto.BaseItemDto, 0, len(u.Libraries))
	for _, lib := range u.Libraries {
		views = append(views, dto.BaseItemDto{
			Id:                strconv.Itoa(lib.ID),
			Name:              lib.Name,
			Type:              "CollectionFolder",
			CollectionType:    "music",
			IsFolder:          true,
			BackdropImageTags: []string{},
		})
	}
	api.ok(w, r, dto.QueryResult{Items: views, TotalRecordCount: len(views), StartIndex: 0})
}

func (api *Router) getCurrentUser(w http.ResponseWriter, r *http.Request) {
	u, _ := request.UserFrom(r.Context())
	api.ok(w, r, userToDto(&u, api.serverName()))
}

func (api *Router) getPublicUsers(w http.ResponseWriter, r *http.Request) {
	api.ok(w, r, []dto.UserDto{}) // manual login only
}

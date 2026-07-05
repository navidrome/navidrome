package jellyfin

import (
	"net/http"

	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
)

const musicViewID = "music"

func (api *Router) getUserViews(w http.ResponseWriter, r *http.Request) {
	view := dto.BaseItemDto{
		Name:              api.serverName(),
		Id:                musicViewID,
		Type:              "CollectionFolder",
		CollectionType:    "music",
		IsFolder:          true,
		BackdropImageTags: []string{},
	}
	api.ok(w, r, dto.QueryResult{Items: []dto.BaseItemDto{view}, TotalRecordCount: 1, StartIndex: 0})
}

func (api *Router) getCurrentUser(w http.ResponseWriter, r *http.Request) {
	u, _ := request.UserFrom(r.Context())
	api.ok(w, r, userToDto(&u, api.serverName()))
}

func (api *Router) getPublicUsers(w http.ResponseWriter, r *http.Request) {
	api.ok(w, r, []dto.UserDto{}) // manual login only
}

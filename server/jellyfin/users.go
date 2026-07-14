package jellyfin

import (
	"net/http"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
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

// getPublicUsers advertises the users named in Jellyfin.ExposedPublicUsers for a client login
// picker. The route is unauthenticated, so it lists only the configured allowlist (never the full
// user table) and returns a minimal DTO — no Policy/Configuration, which would leak admin status.
func (api *Router) getPublicUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serverID := api.serverID(ctx)
	seen := make(map[string]bool)
	users := []dto.UserDto{}
	for name := range strings.SplitSeq(conf.Server.Jellyfin.ExposedPublicUsers, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		usr, err := api.ds.User(ctx).FindByUsername(name)
		if err != nil {
			log.Warn(ctx, "Jellyfin API: configured public user not found", "username", name, err)
			continue
		}
		users = append(users, dto.UserDto{
			Name:        usr.UserName,
			Id:          usr.ID,
			ServerId:    serverID,
			HasPassword: true,
		})
	}
	api.ok(w, r, users)
}

package jellyfin

import (
	"net/http"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/server/imghttp"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
)

// getUserImage is registered unauthenticated, like Jellyfin's own GET /UserImage, so a login picker
// can show avatars; anonymous callers are limited to the ExposedPublicUsers allowlist.
func (api *Router) getUserImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// This route skips the authenticate middleware, so a caller's token (if any) is resolved directly.
	caller, authenticated := api.userFromToken(r)

	rawID := r.URL.Query().Get("userid")
	if rawID == "" {
		if !authenticated {
			http.Error(w, "UserId is required if unauthenticated", http.StatusBadRequest)
			return
		}
		rawID = dto.EncodeID(caller.ID)
	}
	id, ok := dto.DecodeID(rawID)
	if !ok {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	usr, err := api.ds.User(ctx).Get(id)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	if !authenticated && !isPublicUser(usr.UserName) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if !imghttp.ServeUserAvatar(w, r, usr) {
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

// publicUsernames splits and normalizes the raw ExposedPublicUsers config: comma-separated, trimmed,
// skipping empty entries. Shared with getPublicUsers so the two allowlist checks can't drift apart.
func publicUsernames() []string {
	var names []string
	for name := range strings.SplitSeq(conf.Server.Jellyfin.ExposedPublicUsers, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func isPublicUser(username string) bool {
	for _, name := range publicUsernames() {
		if strings.EqualFold(name, username) {
			return true
		}
	}
	return false
}

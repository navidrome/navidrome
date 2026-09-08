package jellyfin

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
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

var avatarExtByContentType = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/jpg":  ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// targetUser resolves the userid param, defaulting to the caller, and applies the
// self-or-admin write rule plus the feature flag (which refuses everyone, admins included).
func (api *Router) targetUser(w http.ResponseWriter, r *http.Request) (*model.User, bool) {
	ctx := r.Context()
	caller, _ := request.UserFrom(ctx)
	id := caller.ID
	if raw := r.URL.Query().Get("userid"); raw != "" {
		decoded, ok := dto.DecodeID(raw)
		if !ok {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return nil, false
		}
		id = decoded
	}
	if !conf.Server.EnableUserAvatarUpload || (!caller.IsAdmin && caller.ID != id) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return nil, false
	}
	usr, err := api.ds.User(ctx).Get(id)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return nil, false
	}
	return usr, true
}

func (api *Router) postUserImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	usr, ok := api.targetUser(w, r)
	if !ok {
		return
	}
	mimeType := strings.TrimSpace(strings.SplitN(r.Header.Get("Content-Type"), ";", 2)[0])
	ext, known := avatarExtByContentType[strings.ToLower(mimeType)]
	if !known {
		http.Error(w, "Incorrect ContentType.", http.StatusBadRequest)
		return
	}

	// Jellyfin clients base64-encode the wire body (4/3 bigger), so the read cap allows for inflation.
	limit := artwork.MaxImageUploadSize()
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit*4/3+4))
	if err != nil {
		http.Error(w, "file too large", http.StatusBadRequest)
		return
	}
	imgBytes, err := decodeImageBody(body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	filename, err := api.imgUpload.SetAvatar(ctx, usr.ID, usr.UserName, usr.UploadedImagePath(), bytes.NewReader(imgBytes), ext)
	if err != nil {
		log.Error(ctx, "Jellyfin API: could not save avatar", "user", usr.UserName, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := api.ds.User(ctx).UpdateImage(usr.ID, filename); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *Router) deleteUserImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	usr, ok := api.targetUser(w, r)
	if !ok {
		return
	}
	if err := api.imgUpload.RemoveImage(ctx, usr.UploadedImagePath()); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := api.ds.User(ctx).UpdateImage(usr.ID, ""); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

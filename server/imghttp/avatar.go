package imghttp

import (
	"net/http"
	"os"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// ServeUserAvatar writes the user's uploaded avatar, reporting whether it answered the request;
// when false the caller must fall back. http.ServeContent gives correct ETag/If-None-Match handling.
func ServeUserAvatar(w http.ResponseWriter, r *http.Request, u *model.User) bool {
	path := u.UploadedImagePath()
	if path == "" {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		log.Warn(r.Context(), "Could not open user avatar", "user", u.UserName, err)
		return false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		log.Warn(r.Context(), "Could not stat user avatar", "user", u.UserName, err)
		return false
	}

	w.Header().Set("ETag", `"`+u.AvatarTag()+`"`)
	w.Header().Set("Cache-Control", "private, no-cache")
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
	return true
}

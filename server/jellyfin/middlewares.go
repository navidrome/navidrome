package jellyfin

import (
	"net/http"
	"regexp"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/request"
)

type embyAuth struct {
	Client, Device, DeviceId, Version, Token string
}

var embyAuthField = regexp.MustCompile(`(\w+)="([^"]*)"`)

// parseEmbyAuth reads the MediaBrowser/Emby authorization header, e.g.
// `MediaBrowser Client="Finamp", Device="Pixel", DeviceId="abc", Version="1.0", Token="jwt"`.
func parseEmbyAuth(r *http.Request) embyAuth {
	var a embyAuth
	h := r.Header.Get("X-Emby-Authorization")
	if h == "" {
		h = r.Header.Get("Authorization")
	}
	for _, m := range embyAuthField.FindAllStringSubmatch(h, -1) {
		switch m[1] {
		case "Client":
			a.Client = m[2]
		case "Device":
			a.Device = m[2]
		case "DeviceId":
			a.DeviceId = m[2]
		case "Version":
			a.Version = m[2]
		case "Token":
			a.Token = m[2]
		}
	}
	return a
}

func tokenFromRequest(r *http.Request) string {
	if t := r.Header.Get("X-Emby-Token"); t != "" {
		return t
	}
	if t := r.Header.Get("X-MediaBrowser-Token"); t != "" {
		return t
	}
	if t := parseEmbyAuth(r).Token; t != "" {
		return t
	}
	return r.URL.Query().Get("api_key")
}

func (api *Router) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		token := tokenFromRequest(r)
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		claims, err := auth.Validate(token)
		if err != nil || claims.Subject == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		usr, err := api.ds.User(ctx).FindByUsername(claims.Subject)
		if err != nil {
			log.Warn(ctx, "Jellyfin API: token subject not found", "user", claims.Subject, err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		ctx = request.WithUser(ctx, *usr)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

package jellyfin

import (
	"net"
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
	if t := r.URL.Query().Get("api_key"); t != "" {
		return t
	}
	// Finamp's just_audio engine fetches direct-file URLs with ?ApiKey= (PascalCase).
	return r.URL.Query().Get("ApiKey")
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

// withPlayer resolves/registers a model.Player for the calling device and injects
// it into the context, mirroring the Subsonic API's getPlayer middleware. Unlike
// Subsonic (which has no stable client-supplied device id and falls back to a
// cookie), Jellyfin clients always send a DeviceId in the auth header, so it's
// used directly as the player id: playback reports from the same install
// consistently resolve to the same player/scrobbling session.
func (api *Router) withPlayer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		a := parseEmbyAuth(r)
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		player, _, err := api.players.Register(ctx, a.DeviceId, a.Client, a.Device, ip)
		if err != nil {
			// Fail open: log and proceed without a player in context, same as Subsonic's
			// getPlayer. Playback reporting handlers degrade gracefully when no player is set.
			log.Warn(ctx, "Jellyfin API: could not register player", "client", a.Client, "device", a.Device, err)
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r.WithContext(request.WithPlayer(ctx, *player)))
	})
}

package jellyfin

import (
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

// normalizeQueryKeys folds every query-parameter key to lowercase so handlers can read params
// case-insensitively, matching real Jellyfin (whose ASP.NET model binding ignores case). Clients
// disagree on casing: Finamp sends PascalCase (ParentId, IncludeItemTypes), while Jellify and the
// official Jellyfin TypeScript SDK send camelCase (parentId, includeItemTypes). A case-sensitive
// read would silently drop one client's filters, sort and paging. The original request is left
// untouched so request logging still shows the casing the client sent; a shallow copy with a
// rewritten query is passed downstream. Handlers therefore read query params by their lowercase
// name. Only keys are folded — values keep their case (e.g. IncludeItemTypes=MusicAlbum).
func normalizeQueryKeys(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		folded := make(url.Values, len(q))
		changed := false
		for k, vs := range q {
			lk := strings.ToLower(k)
			// Append rather than assign: two casings of the same key must merge (url.Values
			// semantics), not have one nondeterministically overwrite the other.
			folded[lk] = append(folded[lk], vs...)
			if lk != k {
				changed = true
			}
		}
		if changed {
			r2 := *r
			u := *r.URL
			u.RawQuery = folded.Encode()
			r2.URL = &u
			r = &r2
		}
		next.ServeHTTP(w, r)
	})
}

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
	// Both api_key and ApiKey are used in the wild (Finamp's just_audio engine fetches direct-file
	// URLs with ?ApiKey=); normalizeQueryKeys has already folded key case, but the two spellings
	// differ by an underscore, not case, so both are still checked.
	if t := r.URL.Query().Get("api_key"); t != "" {
		return t
	}
	return r.URL.Query().Get("apikey")
}

// userFromToken resolves the user identified by the request's token, if any; ok is false for a
// missing/invalid token or an unknown subject. Used by authenticate and by public routes that
// grant extra visibility to an (optional) authenticated caller.
func (api *Router) userFromToken(r *http.Request) (model.User, bool) {
	token := tokenFromRequest(r)
	if token == "" {
		return model.User{}, false
	}
	claims, err := auth.Validate(token)
	if err != nil || claims.Subject == "" {
		return model.User{}, false
	}
	usr, err := api.ds.User(r.Context()).FindByUsername(claims.Subject)
	if err != nil {
		log.Warn(r.Context(), "Jellyfin API: token subject not found", "user", claims.Subject, err)
		return model.User{}, false
	}
	return *usr, true
}

func (api *Router) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		usr, ok := api.userFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := request.WithUser(r.Context(), usr)
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

package jellyfin

import (
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

// throttleStreams bounds how many collection responses stream concurrently, so they can't take every
// connection in the shared DB pool: each holds a cursor, and its connection, for the whole
// client-paced response. Excess requests queue rather than fail. limit <= 0 disables it.
//
// Deliberately chi's ThrottleBacklog and not server.ThrottleBacklog: the latter buffers the entire
// response to release its token early, which is right for artwork but would undo the streaming here.
// chi's panics on a non-positive limit, hence the guard.
func throttleStreams(limit int) func(http.Handler) http.Handler {
	if limit <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	return middleware.ThrottleBacklog(limit, consts.RequestThrottleBacklogLimit, consts.RequestThrottleBacklogTimeout)
}

// caseInsensitivePaths lowercases the request path so chi (case-sensitive) matches the
// lowercase-registered routes; Jellyfin clients route case-insensitively. It lowercases id/param
// segments too, which is safe because every id the API emits — user ids included — is lowercase hex
// (dto.EncodeID).
func caseInsensitivePaths(r chi.Router) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Mounted under a parent, chi matches RouteContext.RoutePath, not r.URL.Path.
		if rctx := chi.RouteContext(req.Context()); rctx != nil && rctx.RoutePath != "" {
			rctx.RoutePath = strings.ToLower(rctx.RoutePath)
		} else {
			req.URL.Path = strings.ToLower(req.URL.Path)
		}
		r.ServeHTTP(w, req)
	})
}

// normalizeQueryKeys folds query-parameter keys to lowercase so handlers can read params
// case-insensitively, matching real Jellyfin. Clients disagree on casing (Finamp sends PascalCase,
// Jellify and the Jellyfin TypeScript SDK camelCase), so a case-sensitive read would drop one
// client's filters, sort and paging. Only keys are folded — values keep their case. The original
// request is left untouched (a rewritten copy goes downstream) so logging shows the client's casing.
func normalizeQueryKeys(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		folded := make(url.Values, len(q))
		changed := false
		for k, vs := range q {
			lk := strings.ToLower(k)
			// Append, don't assign: two casings of the same key must merge, not overwrite.
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

type mediaBrowserAuth struct {
	Client, Device, DeviceId, Version, Token string
}

var mediaBrowserAuthField = regexp.MustCompile(`(\w+)="([^"]*)"`)

// parseMediaBrowserAuth reads the MediaBrowser-scheme authorization header, e.g.
// `MediaBrowser Client="Finamp", Device="Pixel", DeviceId="abc", Version="1.0", Token="jwt"`.
// The recommended Authorization header is preferred, but only when it actually carries
// MediaBrowser data — a reverse proxy may inject Basic/Digest credentials there while the client
// sends the deprecated X-Emby-Authorization. Field values are URL-decoded: Jellify (@jellyfin/sdk)
// percent-encodes them (Device="Pixel%208%20Pro"), while Finamp sends them raw; unescapeField
// leaves a raw value untouched.
func parseMediaBrowserAuth(r *http.Request) mediaBrowserAuth {
	if a, ok := parseAuthHeader(r.Header.Get("Authorization")); ok {
		return a
	}
	a, _ := parseAuthHeader(r.Header.Get("X-Emby-Authorization"))
	return a
}

// parseAuthHeader extracts the MediaBrowser fields from one header value; ok reports whether the
// value uses the MediaBrowser scheme ("Emby" is the legacy spelling real Jellyfin also accepts).
func parseAuthHeader(h string) (mediaBrowserAuth, bool) {
	var a mediaBrowserAuth
	scheme, params, found := strings.Cut(h, " ")
	if !found || (!strings.EqualFold(scheme, "MediaBrowser") && !strings.EqualFold(scheme, "Emby")) {
		return a, false
	}
	for _, m := range mediaBrowserAuthField.FindAllStringSubmatch(params, -1) {
		switch m[1] {
		case "Client":
			a.Client = unescapeField(m[2])
		case "Device":
			a.Device = unescapeField(m[2])
		case "DeviceId":
			a.DeviceId = unescapeField(m[2])
		case "Version":
			a.Version = unescapeField(m[2])
		case "Token":
			a.Token = unescapeField(m[2])
		}
	}
	return a, true
}

// unescapeField percent-decodes a header field value, falling back to the raw value when it isn't
// valid encoding (Finamp sends raw values that may contain a literal '%'). PathUnescape, not
// QueryUnescape, so a literal '+' in a value is preserved rather than turned into a space.
func unescapeField(v string) string {
	if decoded, err := url.PathUnescape(v); err == nil {
		return decoded
	}
	return v
}

// tokenFromRequest prefers the recommended Authorization scheme; the rest are legacy spellings
// deprecated by Jellyfin but still sent by clients.
func tokenFromRequest(r *http.Request) string {
	if t := parseMediaBrowserAuth(r).Token; t != "" {
		return t
	}
	if t := r.Header.Get("X-Emby-Token"); t != "" {
		return t
	}
	if t := r.Header.Get("X-MediaBrowser-Token"); t != "" {
		return t
	}
	// api_key and apikey differ by an underscore, not case, so normalizeQueryKeys' folding doesn't
	// merge them; both are checked (Finamp's just_audio engine fetches direct-file URLs with ?ApiKey=).
	if t := r.URL.Query().Get("api_key"); t != "" {
		return t
	}
	return r.URL.Query().Get("apikey")
}

// userFromToken resolves the user for the request's token; ok is false for a missing/invalid token
// or unknown subject. Used by authenticate and by public routes that optionally identify the caller.
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

// withPlayer resolves/registers a model.Player for the calling device into the context, mirroring
// Subsonic's getPlayer. Jellyfin clients always send a DeviceId in the auth header (unlike Subsonic),
// so it's used directly as the player id and reports from the same install share a player/scrobbling
// session.
func (api *Router) withPlayer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if api.players == nil { // fail open when players isn't wired (e.g. in unit tests)
			next.ServeHTTP(w, r)
			return
		}
		ctx := r.Context()
		a := parseMediaBrowserAuth(r)
		// Skip registration when the request can't identify a client (no X-Emby-Authorization, e.g.
		// the /socket handshake that authenticates via ?api_key= only). Otherwise Register would
		// create a junk player with an empty name (" []").
		if a.Client == "" && a.DeviceId == "" {
			next.ServeHTTP(w, r)
			return
		}
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		player, trc, err := api.players.Register(ctx, a.DeviceId, a.Client, a.Device, ip)
		if err != nil {
			// Fail open, like Subsonic's getPlayer: proceed without a player; reporting handlers
			// degrade gracefully.
			log.Warn(ctx, "Jellyfin API: could not register player", "client", a.Client, "device", a.Device, err)
			next.ServeHTTP(w, r)
			return
		}
		ctx = request.WithPlayer(ctx, *player)
		// Like Subsonic's getPlayer: the forced transcoding must reach ResolveRequest's override.
		if trc != nil {
			ctx = request.WithTranscoding(ctx, *trc)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

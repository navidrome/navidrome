package jellyfin

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type Router struct {
	http.Handler
	ds               model.DataStore
	artwork          artwork.Artwork
	streamer         stream.MediaStreamer
	transcodeDecider stream.TranscodeDecider
	players          core.Players
	scrobbler        scrobbler.PlayTracker
	playlists        playlists.Playlists
	serverIDOnce     sync.Once
	serverIDVal      string
	// canon maps each lower-cased literal route segment to the case it was registered with
	// (e.g. "audio" -> "Audio"). Populated by routes(); exposed here mainly so tests can
	// verify normalizeCase against the real, registered route table.
	canon map[string]string
}

func New(ds model.DataStore, artwork artwork.Artwork, streamer stream.MediaStreamer,
	transcodeDecider stream.TranscodeDecider, players core.Players,
	scrobbler scrobbler.PlayTracker, playlists playlists.Playlists) *Router {
	r := &Router{
		ds: ds, artwork: artwork, streamer: streamer, transcodeDecider: transcodeDecider,
		players: players, scrobbler: scrobbler, playlists: playlists,
	}
	r.Handler = r.routes()
	return r
}

func (api *Router) routes() http.Handler {
	inner := chi.NewRouter()

	// Public (no auth): handshake + login.
	inner.Get("/System/Info/Public", api.getPublicSystemInfo)
	inner.Get("/System/Ping", api.ping)
	inner.Post("/System/Ping", api.ping)
	inner.Get("/QuickConnect/Enabled", api.quickConnectEnabled)
	inner.Post("/Users/AuthenticateByName", api.authenticateByName)
	inner.Get("/Users/Public", api.getPublicUsers)

	// Images are intentionally fully public and do not require authentication: artwork isn't
	// sensitive media content, and this matches Jellyfin's lenient image handling.
	inner.Get("/Items/{itemId}/Images/{type}", api.getItemImage)
	inner.Get("/Items/{itemId}/Images/{type}/{index}", api.getItemImage)

	inner.Group(func(r chi.Router) {
		r.Use(api.authenticate)
		r.Get("/UserViews", api.getUserViews)
		r.Get("/Users/{userId}/Views", api.getUserViews)
		r.Get("/Users/Me", api.getCurrentUser)
		r.Get("/Users/{userId}", api.getCurrentUser)

		r.Get("/Items", api.getItems)
		r.Get("/Users/{userId}/Items", api.getItems)
		r.Get("/Items/{itemId}", api.getItem)
		r.Get("/Users/{userId}/Items/{itemId}", api.getItem)
		r.Get("/Users/{userId}/Items/Latest", api.getLatest)

		r.Post("/Users/{userId}/FavoriteItems/{itemId}", api.markFavorite)
		r.Delete("/Users/{userId}/FavoriteItems/{itemId}", api.unmarkFavorite)
		r.Post("/Users/{userId}/Items/{itemId}/Rating", api.setRating)
		r.Delete("/Users/{userId}/Items/{itemId}/Rating", api.removeRating)

		r.Get("/Artists", api.getArtists)
		r.Get("/Artists/AlbumArtists", api.getArtists)
		r.Get("/Genres", api.getGenres)
		r.Get("/MusicGenres", api.getGenres)

		r.Post("/Playlists", api.createPlaylist)
		r.Get("/Playlists/{playlistId}/Items", api.getPlaylistItems)
		r.Post("/Playlists/{playlistId}/Items", api.addToPlaylist)
		r.Delete("/Playlists/{playlistId}/Items", api.removeFromPlaylist)

		r.Get("/Audio/{itemId}/stream", api.streamAudio)
		r.Get("/Audio/{itemId}/stream.{container}", api.streamAudio)
		r.Get("/Audio/{itemId}/universal", api.streamAudio)
		r.Get("/Items/{itemId}/PlaybackInfo", api.getPlaybackInfo)
		r.Post("/Items/{itemId}/PlaybackInfo", api.getPlaybackInfo)

		// Playback reports carry only the caller's own play data (see reportPlaybackStart
		// doc comment), so no library-access gate is needed here.
		r.Group(func(r chi.Router) {
			r.Use(api.withPlayer)
			r.Post("/Sessions/Playing", api.reportPlaybackStart)
			r.Post("/Sessions/Playing/Progress", api.reportPlaybackProgress)
			r.Post("/Sessions/Playing/Stopped", api.reportPlaybackStopped)
		})
		r.Post("/Sessions/Capabilities", api.postCapabilities)
		r.Post("/Sessions/Capabilities/Full", api.postCapabilities)
	})

	// Logged at Debug (not Warn/Error) because a real client probing for optional/legacy
	// endpoints is expected traffic, not a problem; this just surfaces what's missing.
	inner.NotFound(api.notFound)
	inner.MethodNotAllowed(api.notFound)

	api.canon = canonicalRouteSegments(inner)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		normalizeRequestPath(r, api.canon)
		inner.ServeHTTP(w, r)
	})
}

// canonicalRouteSegments walks every registered route and records, for each literal (non-param)
// "/"-separated segment, the case it was registered with, keyed by its lower-cased form (e.g.
// "audio" -> "Audio"). Chi param placeholders (e.g. "{itemId}") are skipped so id segments are
// never treated as literals.
func canonicalRouteSegments(router chi.Router) map[string]string {
	canon := map[string]string{}
	_ = chi.Walk(router, func(_, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		for _, seg := range strings.Split(route, "/") {
			if seg == "" || strings.Contains(seg, "{") {
				continue
			}
			canon[strings.ToLower(seg)] = seg
		}
		return nil
	})
	return canon
}

// normalizeRequestPath rewrites literal path segments to the case routes were registered with,
// since real Jellyfin clients route case-insensitively while chi does not. It must run before
// chi's matching. When this router is mounted under a parent (as it is in production via
// server.MountRouter), chi has already stripped the mount prefix and matches against
// RouteContext.RoutePath rather than r.URL.Path, so that's what must be normalized here.
func normalizeRequestPath(r *http.Request, canon map[string]string) {
	if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePath != "" {
		rctx.RoutePath = normalizeCase(rctx.RoutePath, canon)
		return
	}
	r.URL.Path = normalizeCase(r.URL.Path, canon)
}

// normalizeCase rewrites each "/"-separated literal segment of path to the case it was
// registered with in canon. Segments with no match (e.g. case-sensitive ids) are left untouched.
func normalizeCase(path string, canon map[string]string) string {
	segs := strings.Split(path, "/")
	for i, seg := range segs {
		if canonical, ok := canon[strings.ToLower(seg)]; ok {
			segs[i] = canonical
		}
	}
	return strings.Join(segs, "/")
}

// ok writes payload as JSON.
func (api *Router) ok(w http.ResponseWriter, r *http.Request, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Error(r.Context(), "Jellyfin API: error encoding response", err)
	}
}

// notFound handles both unmatched routes and unsupported methods on known routes, so any
// endpoint a real client needs that we haven't implemented shows up in the logs instead of
// silently confusing the client with chi's default plain-text 404/405.
func (api *Router) notFound(w http.ResponseWriter, r *http.Request) {
	log.Debug(r.Context(), "Jellyfin API: unhandled route", "method", r.Method, "path", r.URL.Path)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{}`))
}

// internalError logs the real error server-side and writes a generic 500 response, so ffmpeg
// output, file paths or other internal detail in err never reaches the client.
func (api *Router) internalError(w http.ResponseWriter, r *http.Request, err error) {
	log.Error(r.Context(), "Jellyfin API: internal error", "method", r.Method, "path", r.URL.Path, err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

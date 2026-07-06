package jellyfin

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
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
	provider         external.Provider
	serverIDOnce     sync.Once
	serverIDVal      string
}

func New(ds model.DataStore, artwork artwork.Artwork, streamer stream.MediaStreamer,
	transcodeDecider stream.TranscodeDecider, players core.Players,
	scrobbler scrobbler.PlayTracker, playlists playlists.Playlists, provider external.Provider) *Router {
	r := &Router{
		ds: ds, artwork: artwork, streamer: streamer, transcodeDecider: transcodeDecider,
		players: players, scrobbler: scrobbler, playlists: playlists, provider: provider,
	}
	r.Handler = r.routes()
	return r
}

func (api *Router) routes() http.Handler {
	inner := chi.NewRouter()

	// Read every query parameter case-insensitively, like real Jellyfin. Must precede all routes
	// so both public and authenticated handlers (and the api_key auth check) see folded keys.
	inner.Use(normalizeQueryKeys)

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
		r.Delete("/Items/{itemId}", api.deleteItem)
		r.Get("/Users/{userId}/Items/Latest", api.getLatest)

		r.Post("/Users/{userId}/FavoriteItems/{itemId}", api.markFavorite)
		r.Delete("/Users/{userId}/FavoriteItems/{itemId}", api.unmarkFavorite)
		r.Post("/Users/{userId}/Items/{itemId}/Rating", api.setRating)
		r.Delete("/Users/{userId}/Items/{itemId}/Rating", api.removeRating)

		// Per-item play/favorite/rating state. Jellify fetches the /UserItems form per item to
		// render played/favourite indicators; the /Users/{userId}/Items form is the legacy spelling.
		r.Get("/UserItems/{itemId}/UserData", api.getUserItemData)
		r.Get("/Users/{userId}/Items/{itemId}/UserData", api.getUserItemData)

		r.Get("/Artists", api.getArtists)
		r.Get("/Artists/AlbumArtists", api.getAlbumArtists)
		r.Get("/Artists/{itemId}/Similar", api.getSimilarArtists)
		r.Get("/Items/{itemId}/Similar", api.getSimilarItems)
		r.Get("/Genres", api.getGenres)
		r.Get("/MusicGenres", api.getGenres)

		r.Post("/Playlists", api.createPlaylist)
		r.Get("/Playlists/{playlistId}", api.getPlaylist)
		r.Post("/Playlists/{playlistId}", api.updatePlaylist)
		r.Get("/Playlists/{playlistId}/Items", api.getPlaylistItems)
		r.Post("/Playlists/{playlistId}/Items", api.addToPlaylist)
		r.Delete("/Playlists/{playlistId}/Items", api.removeFromPlaylist)
		r.Get("/Playlists/{playlistId}/Users", api.getPlaylistUsers)
		r.Get("/Playlists/{playlistId}/Users/{userId}", api.getPlaylistUser)

		// Cover upload/delete: only playlists are writable through this API (see
		// postItemImage); the GET routes above stay public and unauthenticated.
		r.Post("/Items/{itemId}/Images/{type}", api.postItemImage)
		r.Delete("/Items/{itemId}/Images/{type}", api.deleteItemImage)

		r.Get("/Audio/{itemId}/stream", api.streamAudio)
		r.Get("/Audio/{itemId}/stream.{container}", api.streamAudio)
		r.Get("/Audio/{itemId}/universal", api.streamAudio)
		r.Get("/Items/{itemId}/PlaybackInfo", api.getPlaybackInfo)
		r.Post("/Items/{itemId}/PlaybackInfo", api.getPlaybackInfo)
		// Direct-file endpoints: some clients (e.g. Finamp's just_audio) fetch playback audio
		// here instead of /Audio/{id}/stream after PlaybackInfo; /Download reuses the same
		// direct-play handler since Jellyfin serves the same original file for both.
		r.Get("/Items/{itemId}/File", api.streamFile)
		r.Get("/Items/{itemId}/Download", api.streamFile)

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

		// Real-time clients (e.g. Finamp) open this right after login; without it they
		// 404-loop-reconnect instead of settling into a working session.
		r.Get("/socket", api.handleSocket)
	})

	// Logged at Debug (not Warn/Error) because a real client probing for optional/legacy
	// endpoints is expected traffic, not a problem; this just surfaces what's missing.
	inner.NotFound(api.notFound)
	inner.MethodNotAllowed(api.notFound)

	// Real Jellyfin clients route case-insensitively; chi does not.
	return server.CaseInsensitivePaths(inner)
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

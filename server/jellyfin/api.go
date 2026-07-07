package jellyfin

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"golang.org/x/sync/singleflight"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
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
	similarFlight    singleflight.Group
	serverIDMu       sync.Mutex
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

	// Read query params case-insensitively, like real Jellyfin. Must precede all routes so every
	// handler and the api_key check see folded keys.
	inner.Use(normalizeQueryKeys)

	// Public (no auth): handshake + login.
	inner.Get("/System/Info/Public", api.getPublicSystemInfo)
	inner.Get("/System/Ping", api.ping)
	inner.Post("/System/Ping", api.ping)
	inner.Get("/QuickConnect/Enabled", api.quickConnectEnabled)
	// Rate-limit the password login, mirroring the native /auth/login: it's an unauthenticated
	// brute-force surface, so it must share the same per-IP throttle when one is configured.
	if conf.Server.AuthRequestLimit > 0 {
		limiter := httprate.LimitByIP(conf.Server.AuthRequestLimit, conf.Server.AuthWindowLength)
		inner.With(limiter).Post("/Users/AuthenticateByName", api.authenticateByName)
	} else {
		inner.Post("/Users/AuthenticateByName", api.authenticateByName)
	}
	inner.Get("/Users/Public", api.getPublicUsers)

	// Images are intentionally public: artwork isn't sensitive, matching Jellyfin's image handling.
	inner.Get("/Items/{itemId}/Images/{type}", api.getItemImage)
	inner.Get("/Items/{itemId}/Images/{type}/{index}", api.getItemImage)

	inner.Group(func(r chi.Router) {
		r.Use(api.authenticate)
		// Register/refresh the calling device as a player on every authenticated request, like
		// Subsonic's getPlayer, so Jellyfin clients show up in the players list (and scrobbling has a
		// player) even before the first playback report.
		r.Use(api.withPlayer)
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

		// /UserFavoriteItems is the current @jellyfin/sdk spelling (Jellify); the
		// /Users/{userId}/FavoriteItems form is the legacy one Finamp still uses.
		r.Post("/UserFavoriteItems/{itemId}", api.markFavorite)
		r.Delete("/UserFavoriteItems/{itemId}", api.unmarkFavorite)
		r.Post("/Users/{userId}/FavoriteItems/{itemId}", api.markFavorite)
		r.Delete("/Users/{userId}/FavoriteItems/{itemId}", api.unmarkFavorite)
		r.Post("/Users/{userId}/Items/{itemId}/Rating", api.setRating)
		r.Delete("/Users/{userId}/Items/{itemId}/Rating", api.removeRating)

		// Per-item play/favorite/rating state. Jellify uses the /UserItems form;
		// /Users/{userId}/Items is the legacy spelling.
		r.Get("/UserItems/{itemId}/UserData", api.getUserItemData)
		r.Get("/Users/{userId}/Items/{itemId}/UserData", api.getUserItemData)

		r.Get("/Artists", api.getArtists)
		r.Get("/Artists/AlbumArtists", api.getAlbumArtists)
		r.Get("/Artists/{itemId}/Similar", api.getSimilarArtists)
		r.Get("/Items/{itemId}/Similar", api.getSimilarItems)
		r.Get("/Items/{itemId}/InstantMix", api.getInstantMix)
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

		// Cover upload/delete: only playlists are writable (see postItemImage); the GET routes
		// above stay public.
		r.Post("/Items/{itemId}/Images/{type}", api.postItemImage)
		r.Delete("/Items/{itemId}/Images/{type}", api.deleteItemImage)

		r.Get("/Audio/{itemId}/stream", api.streamAudio)
		r.Get("/Audio/{itemId}/stream.{container}", api.streamAudio)
		r.Get("/Audio/{itemId}/universal", api.streamAudio)
		r.Get("/Audio/{itemId}/main.m3u8", api.streamHls)
		r.Get("/Items/{itemId}/PlaybackInfo", api.getPlaybackInfo)
		r.Post("/Items/{itemId}/PlaybackInfo", api.getPlaybackInfo)
		// Direct-file endpoints: some clients (Finamp's just_audio) fetch here instead of
		// /Audio/{id}/stream; /Download reuses the direct-play handler as Jellyfin serves the same file.
		r.Get("/Items/{itemId}/File", api.streamFile)
		r.Get("/Items/{itemId}/Download", api.streamFile)

		r.Post("/Sessions/Playing", api.reportPlaybackStart)
		r.Post("/Sessions/Playing/Progress", api.reportPlaybackProgress)
		r.Post("/Sessions/Playing/Stopped", api.reportPlaybackStopped)
		r.Post("/Sessions/Capabilities", api.postCapabilities)
		r.Post("/Sessions/Capabilities/Full", api.postCapabilities)

		// Real-time clients (e.g. Finamp) open this right after login; without it they 404-loop-reconnect.
		r.Get("/socket", api.handleSocket)
	})

	// Logged at Debug, not Warn/Error: clients probing for optional/legacy endpoints is expected
	// traffic, and this just surfaces what's missing.
	inner.NotFound(api.notFound)
	inner.MethodNotAllowed(api.notFound)

	// Real Jellyfin clients route case-insensitively; chi does not.
	return caseInsensitivePaths(inner)
}

// ok writes payload as JSON, stamping ServerId on any item(s) in it — real Jellyfin always sets it,
// and it's the same value for every item, so it's applied here rather than threaded through mappers.
func (api *Router) ok(w http.ResponseWriter, r *http.Request, payload any) {
	switch p := payload.(type) {
	case dto.QueryResult:
		api.stampServerID(r.Context(), p.Items)
	case []dto.BaseItemDto:
		api.stampServerID(r.Context(), p)
	case dto.BaseItemDto:
		p.ServerId = api.serverID(r.Context())
		payload = p
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Error(r.Context(), "Jellyfin API: error encoding response", err)
	}
}

func (api *Router) stampServerID(ctx context.Context, items []dto.BaseItemDto) {
	sid := api.serverID(ctx)
	for i := range items {
		items[i].ServerId = sid
	}
}

// notFound handles unmatched routes and unsupported methods, logging them so unimplemented
// endpoints surface instead of returning chi's default plain-text 404/405.
func (api *Router) notFound(w http.ResponseWriter, r *http.Request) {
	log.Debug(r.Context(), "Jellyfin API: unhandled route", "method", r.Method, "path", r.URL.Path)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{}`))
}

// internalError logs the real error and writes a generic 500, so internal detail (ffmpeg output,
// file paths) never reaches the client.
func (api *Router) internalError(w http.ResponseWriter, r *http.Request, err error) {
	log.Error(r.Context(), "Jellyfin API: internal error", "method", r.Method, "path", r.URL.Path, err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

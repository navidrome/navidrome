package jellyfin

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"golang.org/x/sync/singleflight"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/lyrics"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/core/sonic"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/utils/cache"
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
	sonic            sonic.Engine
	lyrics           lyrics.Lyrics
	lyricsCache      cache.SimpleCache[string, model.LyricList]
	similarFlight    singleflight.Group
	serverIDMu       sync.Mutex
	serverIDVal      string
}

func New(ds model.DataStore, artwork artwork.Artwork, streamer stream.MediaStreamer,
	transcodeDecider stream.TranscodeDecider, players core.Players,
	scrobbler scrobbler.PlayTracker, playlists playlists.Playlists, provider external.Provider,
	sonicSvc sonic.Engine, lyricsSvc lyrics.Lyrics) *Router {
	r := &Router{
		ds: ds, artwork: artwork, streamer: streamer, transcodeDecider: transcodeDecider,
		players: players, scrobbler: scrobbler, playlists: playlists, provider: provider,
		sonic: sonicSvc, lyrics: lyricsSvc,
		lyricsCache: cache.NewSimpleCache[string, model.LyricList](cache.Options{
			SizeLimit:  1000,
			DefaultTTL: 5 * time.Minute,
		}),
	}
	r.Handler = r.routes()
	return r
}

func (api *Router) routes() http.Handler {
	inner := chi.NewRouter()

	// Read query params case-insensitively, like real Jellyfin. Must precede all routes so every
	// handler and the api_key check see folded keys.
	inner.Use(normalizeQueryKeys)

	// Routes are lowercase; caseInsensitivePaths lowercases the request path. Keep new routes lowercase.

	// Public (no auth): handshake + login.
	inner.Get("/system/info/public", api.getPublicSystemInfo)
	inner.Get("/system/ping", api.ping)
	inner.Post("/system/ping", api.ping)
	inner.Get("/quickconnect/enabled", api.quickConnectEnabled)
	// Rate-limit the password login, mirroring the native /auth/login: it's an unauthenticated
	// brute-force surface, so it must share the same per-IP throttle when one is configured.
	if conf.Server.AuthRequestLimit > 0 {
		limiter := httprate.LimitByIP(conf.Server.AuthRequestLimit, conf.Server.AuthWindowLength)
		inner.With(limiter).Post("/users/authenticatebyname", api.authenticateByName)
	} else {
		inner.Post("/users/authenticatebyname", api.authenticateByName)
	}
	inner.Get("/users/public", api.getPublicUsers)

	// Images are intentionally public: artwork isn't sensitive, matching Jellyfin's image handling.
	// Bound concurrency like Subsonic's getCoverArt: image decode/resize is CPU- and memory-heavy,
	// and an unbounded burst (a client fetching artwork across a large library) can exhaust memory.
	inner.Group(func(r chi.Router) {
		r.Use(server.ThrottleBacklog(conf.Server.DevArtworkMaxRequests, conf.Server.DevArtworkThrottleBacklogLimit,
			conf.Server.DevArtworkThrottleBacklogTimeout))
		r.Get("/items/{itemId}/images/{type}", api.getItemImage)
		r.Get("/items/{itemId}/images/{type}/{index}", api.getItemImage)
	})

	inner.Group(func(r chi.Router) {
		r.Use(api.authenticate)
		// Register/refresh the calling device as a player on every authenticated request, like
		// Subsonic's getPlayer, so Jellyfin clients show up in the players list (and scrobbling has a
		// player) even before the first playback report.
		r.Use(api.withPlayer)
		r.Get("/system/info", api.getSystemInfo)
		r.Get("/userviews", api.getUserViews)
		r.Get("/users/{userId}/views", api.getUserViews)
		r.Get("/users/me", api.getCurrentUser)
		r.Get("/users/{userId}", api.getCurrentUser)

		// Cursor-backed collections: each streams straight from the DB, holding a connection for the
		// whole client-paced response, so enough slow clients would take the entire pool and stall the
		// scanner, scrobbles and the UI. Cap them at half the pool (see conf.MaxOpenConns); excess
		// requests queue rather than fail.
		r.Group(func(r chi.Router) {
			r.Use(throttleStreams(conf.Server.Jellyfin.MaxConcurrentStreams))
			r.Get("/items", api.getItems)
			r.Get("/users/{userId}/items", api.getItems)
			r.Get("/users/{userId}/items/latest", api.getLatest)
			r.Get("/artists", api.getArtists)
			r.Get("/artists/albumartists", api.getAlbumArtists)
			r.Get("/playlists/{playlistId}/items", api.getPlaylistItems)
		})

		r.Get("/items/{itemId}", api.getItem)
		r.Get("/users/{userId}/items/{itemId}", api.getItem)
		r.Delete("/items/{itemId}", api.deleteItem)

		// /UserFavoriteItems is the current @jellyfin/sdk spelling (Jellify); the
		// /Users/{userId}/FavoriteItems form is the legacy one Finamp still uses.
		r.Post("/userfavoriteitems/{itemId}", api.markFavorite)
		r.Delete("/userfavoriteitems/{itemId}", api.unmarkFavorite)
		r.Post("/users/{userId}/favoriteitems/{itemId}", api.markFavorite)
		r.Delete("/users/{userId}/favoriteitems/{itemId}", api.unmarkFavorite)
		r.Post("/users/{userId}/items/{itemId}/rating", api.setRating)
		r.Delete("/users/{userId}/items/{itemId}/rating", api.removeRating)

		// Per-item play/favorite/rating state. Jellify uses the /UserItems form;
		// /Users/{userId}/Items is the legacy spelling.
		r.Get("/useritems/{itemId}/userdata", api.getUserItemData)
		r.Get("/users/{userId}/items/{itemId}/userdata", api.getUserItemData)

		r.Get("/artists/{itemId}/similar", api.getSimilarArtists)
		r.Get("/items/{itemId}/similar", api.getSimilarItems)
		r.Get("/items/{itemId}/instantmix", api.getInstantMix)
		r.Get("/genres", api.getGenres)
		r.Get("/musicgenres", api.getGenres)

		r.Post("/playlists", api.createPlaylist)
		r.Get("/playlists/{playlistId}", api.getPlaylist)
		r.Post("/playlists/{playlistId}", api.updatePlaylist)
		r.Post("/playlists/{playlistId}/items", api.addToPlaylist)
		r.Delete("/playlists/{playlistId}/items", api.removeFromPlaylist)
		r.Get("/playlists/{playlistId}/users", api.getPlaylistUsers)
		r.Get("/playlists/{playlistId}/users/{userId}", api.getPlaylistUser)

		// Cover upload/delete: only playlists are writable (see postItemImage); the GET routes
		// above stay public.
		r.Post("/items/{itemId}/images/{type}", api.postItemImage)
		r.Delete("/items/{itemId}/images/{type}", api.deleteItemImage)

		r.Get("/audio/{itemId}/stream", api.streamAudio)
		r.Get("/audio/{itemId}/stream.{container}", api.streamAudio)
		r.Get("/audio/{itemId}/universal", api.streamAudio)
		r.Get("/audio/{itemId}/main.m3u8", api.streamHls)
		r.Get("/items/{itemId}/playbackinfo", api.getPlaybackInfo)
		r.Post("/items/{itemId}/playbackinfo", api.getPlaybackInfo)
		r.Get("/audio/{itemId}/lyrics", api.getLyrics)
		// Direct-file endpoints: some clients (Finamp's just_audio) fetch here instead of
		// /Audio/{id}/stream; /Download reuses the direct-play handler as Jellyfin serves the same file.
		r.Get("/items/{itemId}/file", api.streamFile)
		r.Get("/items/{itemId}/download", api.streamFile)

		r.Post("/sessions/playing", api.reportPlaybackStart)
		r.Post("/sessions/playing/progress", api.reportPlaybackProgress)
		r.Post("/sessions/playing/stopped", api.reportPlaybackStopped)
		r.Post("/sessions/capabilities", api.postCapabilities)
		r.Post("/sessions/capabilities/full", api.postCapabilities)

		// Real-time clients (e.g. Finamp) open this right after login; without it they 404-loop-reconnect.
		r.Get("/socket", api.handleSocket)

		r.Get("/audiomuseai/info", api.audioMuseInfo)
		r.Get("/audiomuseai/health", api.audioMuseHealth)
		r.Get("/audiomuseai/similar_tracks", api.audioMuseSimilarTracks)
		r.Get("/audiomuseai/find_path", api.audioMuseFindPath)
	})

	// Logged at Debug, not Warn/Error: clients probing for optional/legacy endpoints is expected
	// traffic, and this just surfaces what's missing.
	inner.NotFound(api.notFound)
	inner.MethodNotAllowed(api.notFound)

	// Real Jellyfin clients route case-insensitively; chi does not.
	return caseInsensitivePaths(inner)
}

// ok writes payload as JSON — the single entry point for every handler. Collections are routed to
// the streaming writer, so callers needn't know whether theirs is cursor-backed. ServerId is stamped
// on any item(s): real Jellyfin always sets it, and it's constant per request.
//
// Only /Items/Latest bypasses this, for its bare-array shape (see writeItemsArray).
func (api *Router) ok(w http.ResponseWriter, r *http.Request, payload any) {
	switch p := payload.(type) {
	case itemsResult:
		api.writeItems(w, r, p)
		return
	case dto.QueryResult:
		api.writeItems(w, r, materialized(p))
		return
	case dto.BaseItemDto:
		p.ServerId = api.serverID(r.Context())
		payload = p
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Error(r.Context(), "Jellyfin API: error encoding response", err)
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

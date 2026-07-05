package jellyfin

import (
	"encoding/json"
	"net/http"
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
	r := chi.NewRouter()

	// Public (no auth): handshake + login.
	r.Get("/System/Info/Public", api.getPublicSystemInfo)
	r.Get("/System/Ping", api.ping)
	r.Post("/System/Ping", api.ping)
	r.Get("/QuickConnect/Enabled", api.quickConnectEnabled)
	r.Post("/Users/AuthenticateByName", api.authenticateByName)
	r.Get("/Users/Public", api.getPublicUsers)

	// Images are intentionally public and not library-scoped: artwork isn't sensitive media
	// content, and clients (e.g. Finamp) load it via <img> tags with only ?api_key= in the URL.
	r.Get("/Items/{itemId}/Images/{type}", api.getItemImage)
	r.Get("/Items/{itemId}/Images/{type}/{index}", api.getItemImage)

	r.Group(func(r chi.Router) {
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

		r.Get("/Audio/{itemId}/stream", api.streamAudio)
		r.Get("/Audio/{itemId}/stream.{container}", api.streamAudio)
		r.Get("/Audio/{itemId}/universal", api.streamAudio)
		r.Get("/Items/{itemId}/PlaybackInfo", api.getPlaybackInfo)
		r.Post("/Items/{itemId}/PlaybackInfo", api.getPlaybackInfo)
	})

	return r
}

// ok writes payload as JSON.
func (api *Router) ok(w http.ResponseWriter, r *http.Request, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Error(r.Context(), "Jellyfin API: error encoding response", err)
	}
}

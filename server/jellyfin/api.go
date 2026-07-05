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

	r.Group(func(r chi.Router) {
		r.Use(api.authenticate)
		// authenticated endpoints are registered by later tasks
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

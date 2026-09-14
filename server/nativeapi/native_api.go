package nativeapi

import (
	"context"
	"encoding/json"
	"html"
	"net/http"
	"strconv"
	"time"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/metrics"
	playlistsvc "github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server"
)

// PluginManager defines the interface for plugin management operations.
// This interface is used by the API handlers to enable/disable plugins and update configuration.
type PluginManager interface {
	EnablePlugin(ctx context.Context, id string) error
	DisablePlugin(ctx context.Context, id string) error
	ValidatePluginConfig(ctx context.Context, id, configJSON string) error
	UpdatePluginConfig(ctx context.Context, id, configJSON string) error
	UpdatePluginUsers(ctx context.Context, id, usersJSON string, allUsers bool) error
	UpdatePluginLibraries(ctx context.Context, id, librariesJSON string, allLibraries, allowWriteAccess bool) error
	RescanPlugins(ctx context.Context) error
	UnloadDisabledPlugins(ctx context.Context)
}

type Router struct {
	http.Handler
	ds            model.DataStore
	share         core.Share
	playlists     playlistsvc.Playlists
	insights      metrics.Insights
	libs          core.Library
	users         core.User
	maintenance   core.Maintenance
	pluginManager PluginManager
	imgUpload     artwork.Uploader
	provider      external.Provider
}

func New(ds model.DataStore, share core.Share, playlists playlistsvc.Playlists, insights metrics.Insights, libraryService core.Library, userService core.User, maintenance core.Maintenance, pluginManager PluginManager, imgUpload artwork.Uploader, provider external.Provider) *Router {
	r := &Router{ds: ds, share: share, playlists: playlists, insights: insights, libs: libraryService, users: userService, maintenance: maintenance, pluginManager: pluginManager, imgUpload: imgUpload, provider: provider}
	r.Handler = r.routes()
	return r
}

func (api *Router) routes() http.Handler {
	r := chi.NewRouter()

	// Public
	rx(r, "/translation", newTranslationRepository(), false)

	// Protected
	r.Group(func(r chi.Router) {
		r.Use(server.Authenticator(api.ds))
		r.Use(server.JWTRefresher)
		r.Use(server.UpdateLastAccessMiddleware(api.ds))
		rx(r, "/user", lazyRW(func(ctx context.Context) rest.Repository[model.User] { return api.users.NewRepository(ctx) }), true)
		rx(r, "/song", lazy(func(ctx context.Context) rest.Repository[model.MediaFile] { return api.ds.MediaFile(ctx) }), false)
		rx(r, "/album", lazy(func(ctx context.Context) rest.Repository[model.Album] { return api.ds.Album(ctx) }), false)
		api.addArtistRoute(r)
		rx(r, "/genre", lazy(func(ctx context.Context) rest.Repository[model.Genre] { return api.ds.Genre(ctx) }), false)
		rx(r, "/player", api.ds.Player(), true)
		rx(r, "/transcoding", api.ds.Transcoding(), conf.Server.EnableTranscodingConfig)
		api.addRadioRoute(r)
		rx(r, "/tag", lazy(func(ctx context.Context) rest.Repository[model.Tag] { return api.ds.Tag(ctx) }), false)
		rx(r, "/scrobble", lazy(func(ctx context.Context) rest.Repository[model.Scrobble] { return api.ds.Scrobble(ctx) }), false)
		if conf.Server.EnableSharing {
			rx(r, "/share", lazyRW(func(ctx context.Context) rest.Repository[model.Share] { return api.share.NewRepository(ctx) }), true)
		}

		api.addPlaylistRoute(r)
		api.addPlaylistTrackRoute(r)
		api.addSongPlaylistsRoute(r)
		api.addQueueRoute(r)
		api.addMissingFilesRoute(r)
		api.addKeepAliveRoute(r)
		api.addInsightsRoute(r)

		r.With(adminOnlyMiddleware).Group(func(r chi.Router) {
			api.addInspectRoute(r)
			api.addConfigRoute(r)
			api.addUserLibraryRoute(r)
			api.addPluginRoute(r)
			api.addMetadataRoute(r)
			rx(r, "/library", lazyRW(func(ctx context.Context) rest.Repository[model.Library] { return api.libs.NewRepository(ctx) }), true)
		})
	})

	return r
}

func rx[T any](r chi.Router, pathPrefix string, repo rest.Repository[T], persistable bool) {
	r.Route(pathPrefix, func(r chi.Router) {
		r.Get("/", rest.GetAll(repo))
		if persistable {
			r.Post("/", rest.Post(repo))
		}
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(repo))
			if persistable {
				r.Put("/", rest.Put(repo))
				r.Delete("/", rest.Delete(repo))
			}
		})
	})
}

func (api *Router) addPlaylistRoute(r chi.Router) {
	repo := lazyRW(func(ctx context.Context) rest.Repository[model.Playlist] { return api.playlists.NewRepository(ctx) })

	r.Route("/playlist", func(r chi.Router) {
		r.Get("/", rest.GetAll(repo))
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Content-type") == "application/json" {
				rest.Post(repo)(w, r)
				return
			}
			createPlaylistFromM3U(api.playlists)(w, r)
		})

		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(repo))
			r.Put("/", rest.Put(repo))
			r.Delete("/", rest.Delete(repo))
			r.Post("/image", uploadPlaylistImage(api.playlists))
			r.Delete("/image", deletePlaylistImage(api.playlists))
		})
	})
}

func (api *Router) addPlaylistTrackRoute(r chi.Router) {
	r.Route("/playlist/{playlistId}/tracks", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			getPlaylist(api.playlists)(w, r)
		})
		r.With(server.URLParamsMiddleware).Route("/", func(r chi.Router) {
			r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
				deleteFromPlaylist(api.playlists)(w, r)
			})
			r.Post("/", func(w http.ResponseWriter, r *http.Request) {
				addToPlaylist(api.playlists)(w, r)
			})
		})
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				getPlaylistTrack(api.playlists)(w, r)
			})
			r.Put("/", func(w http.ResponseWriter, r *http.Request) {
				reorderItem(api.playlists)(w, r)
			})
			r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
				deleteFromPlaylist(api.playlists)(w, r)
			})
		})
	})
}

func (api *Router) addSongPlaylistsRoute(r chi.Router) {
	r.With(server.URLParamsMiddleware).Get("/song/{id}/playlists", func(w http.ResponseWriter, r *http.Request) {
		getSongPlaylists(api.playlists)(w, r)
	})
}

func (api *Router) addQueueRoute(r chi.Router) {
	r.Route("/queue", func(r chi.Router) {
		r.Get("/", getQueue(api.ds))
		r.Post("/", saveQueue(api.ds))
		r.Put("/", updateQueue(api.ds))
		r.Delete("/", clearQueue(api.ds))
	})
}

func (api *Router) addMissingFilesRoute(r chi.Router) {
	r.Route("/missing", func(r chi.Router) {
		rx(r, "/", newMissingRepository(api.ds), false)
		r.Delete("/", deleteMissingFiles(api.maintenance))
	})
}

func writeDeleteManyResponse(w http.ResponseWriter, r *http.Request, ids []string) {
	var resp []byte
	var err error
	if len(ids) == 1 {
		resp = []byte(`{"id":"` + html.EscapeString(ids[0]) + `"}`)
	} else {
		resp, err = json.Marshal(&struct {
			Ids []string `json:"ids"`
		}{Ids: ids})
		if err != nil {
			log.Error(r.Context(), "Error marshaling response", "ids", ids, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
	_, err = w.Write(resp) //nolint:gosec
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (api *Router) addInspectRoute(r chi.Router) {
	if conf.Server.Inspect.Enabled {
		r.Group(func(r chi.Router) {
			if conf.Server.Inspect.MaxRequests > 0 {
				log.Debug("Throttling inspect", "maxRequests", conf.Server.Inspect.MaxRequests,
					"backlogLimit", conf.Server.Inspect.BacklogLimit, "backlogTimeout",
					conf.Server.Inspect.BacklogTimeout)
				r.Use(middleware.ThrottleBacklog(conf.Server.Inspect.MaxRequests, conf.Server.Inspect.BacklogLimit, time.Duration(conf.Server.Inspect.BacklogTimeout)))
			}
			r.Get("/inspect", inspect(api.ds))
		})
	}
}

func (api *Router) addConfigRoute(r chi.Router) {
	if conf.Server.DevUIShowConfig {
		r.Get("/config/*", getConfig)
	}
}

func (api *Router) addKeepAliveRoute(r chi.Router) {
	r.Get("/keepalive/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"response":"ok", "id":"keepalive"}`))
	})
}

func (api *Router) addInsightsRoute(r chi.Router) {
	r.Get("/insights/*", func(w http.ResponseWriter, r *http.Request) {
		last, success := api.insights.LastRun(r.Context())
		if conf.Server.EnableInsightsCollector {
			_, _ = w.Write([]byte(`{"id":"insights_status", "lastRun":"` + last.Format("2006-01-02 15:04:05") + `", "success":` + strconv.FormatBool(success) + `}`)) //nolint:gosec
		} else {
			_, _ = w.Write([]byte(`{"id":"insights_status", "lastRun":"disabled", "success":false}`))
		}
	})
}

// Middleware to ensure only admin users can access endpoints
func adminOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := request.UserFrom(r.Context())
		if !ok || !user.IsAdmin {
			http.Error(w, "Access denied: admin privileges required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

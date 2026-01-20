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
	"github.com/navidrome/navidrome/core/metrics"
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
	UpdatePluginLibraries(ctx context.Context, id, librariesJSON string, allLibraries bool) error
	RescanPlugins(ctx context.Context) error
	UnloadDisabledPlugins(ctx context.Context)
}

type Router struct {
	http.Handler
	ds            model.DataStore
	share         core.Share
	playlists     core.Playlists
	insights      metrics.Insights
	libs          core.Library
	users         core.User
	maintenance   core.Maintenance
	pluginManager PluginManager
}

func New(ds model.DataStore, share core.Share, playlists core.Playlists, insights metrics.Insights, libraryService core.Library, userService core.User, maintenance core.Maintenance, pluginManager PluginManager) *Router {
	r := &Router{ds: ds, share: share, playlists: playlists, insights: insights, libs: libraryService, users: userService, maintenance: maintenance, pluginManager: pluginManager}
	r.Handler = r.routes()
	return r
}

func (api *Router) routes() http.Handler {
	r := chi.NewRouter()

	// Public
	api.RX(r, "/translation", newTranslationRepository, false)

	// Protected
	r.Group(func(r chi.Router) {
		r.Use(server.Authenticator(api.ds))
		r.Use(server.JWTRefresher)
		r.Use(server.UpdateLastAccessMiddleware(api.ds))
		api.RX(r, "/user", api.users.NewRepository, true)
		api.R(r, "/song", model.MediaFile{}, false)
		api.R(r, "/album", model.Album{}, false)
		api.R(r, "/artist", model.Artist{}, false)
		api.R(r, "/genre", model.Genre{}, false)
		api.R(r, "/player", model.Player{}, true)
		api.R(r, "/transcoding", model.Transcoding{}, conf.Server.EnableTranscodingConfig)
		api.R(r, "/radio", model.Radio{}, true)
		api.R(r, "/tag", model.Tag{}, true)
		if conf.Server.EnableSharing {
			api.RX(r, "/share", api.share.NewRepository, true)
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
			api.RX(r, "/library", api.libs.NewRepository, true)
		})
	})

	return r
}

func (api *Router) R(r chi.Router, pathPrefix string, model interface{}, persistable bool) {
	constructor := func(ctx context.Context) rest.Repository {
		return api.ds.Resource(ctx, model)
	}
	api.RX(r, pathPrefix, constructor, persistable)
}

func (api *Router) RX(r chi.Router, pathPrefix string, constructor rest.RepositoryConstructor, persistable bool) {
	r.Route(pathPrefix, func(r chi.Router) {
		r.Get("/", rest.GetAll(constructor))
		if persistable {
			r.Post("/", rest.Post(constructor))
		}
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(constructor))
			if persistable {
				r.Put("/", rest.Put(constructor))
				r.Delete("/", rest.Delete(constructor))
			}
		})
	})
}

func (api *Router) addPlaylistRoute(r chi.Router) {
	constructor := func(ctx context.Context) rest.Repository {
		return api.ds.Resource(ctx, model.Playlist{})
	}

	r.Route("/playlist", func(r chi.Router) {
		r.Get("/", rest.GetAll(constructor))
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Content-type") == "application/json" {
				rest.Post(constructor)(w, r)
				return
			}
			createPlaylistFromM3U(api.playlists)(w, r)
		})

		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(constructor))
			r.Put("/", rest.Put(constructor))
			r.Delete("/", rest.Delete(constructor))
		})
	})
}

func (api *Router) addPlaylistTrackRoute(r chi.Router) {
	r.Route("/playlist/{playlistId}/tracks", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			getPlaylist(api.ds)(w, r)
		})
		r.With(server.URLParamsMiddleware).Route("/", func(r chi.Router) {
			r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
				deleteFromPlaylist(api.ds)(w, r)
			})
			r.Post("/", func(w http.ResponseWriter, r *http.Request) {
				addToPlaylist(api.ds)(w, r)
			})
		})
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				getPlaylistTrack(api.ds)(w, r)
			})
			r.Put("/", func(w http.ResponseWriter, r *http.Request) {
				reorderItem(api.ds)(w, r)
			})
			r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
				deleteFromPlaylist(api.ds)(w, r)
			})
		})
	})
}

func (api *Router) addSongPlaylistsRoute(r chi.Router) {
	r.With(server.URLParamsMiddleware).Get("/song/{id}/playlists", func(w http.ResponseWriter, r *http.Request) {
		getSongPlaylists(api.ds)(w, r)
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
		api.RX(r, "/", newMissingRepository(api.ds), false)
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
	_, err = w.Write(resp)
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
			_, _ = w.Write([]byte(`{"id":"insights_status", "lastRun":"` + last.Format("2006-01-02 15:04:05") + `", "success":` + strconv.FormatBool(success) + `}`))
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

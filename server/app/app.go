package app

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/deluan/rest"
	"github.com/go-chi/chi"
	"github.com/go-chi/httprate"
	"github.com/go-chi/jwtauth"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/ui"
)

type Router struct {
	ds     model.DataStore
	mux    http.Handler
	broker events.Broker
}

func New(ds model.DataStore, broker events.Broker) *Router {
	return &Router{ds: ds, broker: broker}
}

func (app *Router) Setup(path string) {
	app.mux = app.routes(path)
}

func (app *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.mux.ServeHTTP(w, r)
}

func (app *Router) routes(path string) http.Handler {
	r := chi.NewRouter()

	if conf.Server.AuthRequestLimit > 0 {
		log.Info("Login rate limit set", "requestLimit", conf.Server.AuthRequestLimit,
			"windowLength", conf.Server.AuthWindowLength)

		rateLimiter := httprate.LimitByIP(conf.Server.AuthRequestLimit, conf.Server.AuthWindowLength)
		r.With(rateLimiter).Post("/login", Login(app.ds))
	} else {
		log.Warn("Login rate limit is disabled! Consider enabling it to be protected against brute-force attacks")

		r.Post("/login", Login(app.ds))
	}

	r.Post("/createAdmin", CreateAdmin(app.ds))

	r.Route("/api", func(r chi.Router) {
		r.Use(mapAuthHeader())
		r.Use(jwtauth.Verifier(auth.TokenAuth))
		r.Use(authenticator(app.ds))
		app.R(r, "/user", model.User{}, true)
		app.ServeMediaFiles(r, "/song", true)
		app.R(r, "/album", model.Album{}, true)
		app.R(r, "/artist", model.Artist{}, true)
		app.R(r, "/player", model.Player{}, true)
		app.R(r, "/playlist", model.Playlist{}, true)
		app.R(r, "/transcoding", model.Transcoding{}, conf.Server.EnableTranscodingConfig)
		app.RX(r, "/translation", newTranslationRepository, false)

		app.addPlaylistTrackRoute(r)

		// Keepalive endpoint to be used to keep the session valid (ex: while playing songs)
		r.Get("/keepalive/*", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"response":"ok"}`)) })

		if conf.Server.DevActivityPanel {
			r.Handle("/events", app.broker)
		}
	})

	// Serve UI app assets
	r.Handle("/", serveIndex(app.ds, ui.Assets()))
	r.Handle("/*", http.StripPrefix(path, http.FileServer(http.FS(ui.Assets()))))

	return r
}

func (app *Router) R(r chi.Router, pathPrefix string, model interface{}, persistable bool) {
	constructor := func(ctx context.Context) rest.Repository {
		return app.ds.Resource(ctx, model)
	}
	app.RX(r, pathPrefix, constructor, persistable)
}

func (app *Router) RX(r chi.Router, pathPrefix string, constructor rest.RepositoryConstructor, persistable bool) {
	r.Route(pathPrefix, func(r chi.Router) {
		r.Get("/", rest.GetAll(constructor))
		if persistable {
			r.Post("/", rest.Post(constructor))
		}
		r.Route("/{id}", func(r chi.Router) {
			r.Use(UrlParams)
			r.Get("/", rest.Get(constructor))
			if persistable {
				r.Put("/", rest.Put(constructor))
				r.Delete("/", rest.Delete(constructor))
			}
		})
	})
}

func (app *Router) ServeMediaFiles(r chi.Router, pathPrefix string, persistable bool) {
	constructor := func(ctx context.Context) rest.Repository {
		return app.ds.Resource(ctx, model.MediaFile{})
	}

	r.Route(pathPrefix, func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			repository := constructor(r.Context())
			options := parseOptions(r.URL.Query())
			var entities interface{}
			var err error
			if err != nil {
				if err = rest.RespondWithError(w, http.StatusInternalServerError, err.Error()); err != nil {
					log.Error(err)
				}
				return
			}
			if r.Context().Value(request.User).(model.User).IsAdmin {
				var tEntities interface{}
				tEntities, err = repository.ReadAll(options)
				mediaFiles := tEntities.(model.MediaFiles)
				adminMediaFiles := make(model.AdminMediaFiles, len(mediaFiles))
				for i, v := range mediaFiles {
					adminMediaFiles[i] = model.AdminMediaFile(v)
				}
				entities = adminMediaFiles
			} else {
				entities, err = repository.ReadAll(options)
			}
			if err != nil {
				if err = rest.RespondWithError(w, http.StatusInternalServerError, err.Error()); err != nil {
					log.Error(err)
				}
				return
			}
			count, _ := repository.Count(options)
			w.Header().Set("X-Total-Count", strconv.FormatInt(count, 10))
			if err = rest.RespondWithJSON(w, 200, &entities); err != nil {
				log.Error(err)
			}
		})
		if persistable {
			r.Post("/", rest.Post(constructor))
		}
		r.Route("/{id}", func(r chi.Router) {
			r.Use(UrlParams)
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				id := r.URL.Query().Get(":id")
				repository := constructor(r.Context())
				entity, err := repository.Read(id)
				if err == rest.ErrNotFound {
					msg := fmt.Sprintf("%s(id:%s) not found", repository.EntityName(), id)
					if err = rest.RespondWithError(w, http.StatusInternalServerError, msg); err != nil {
						log.Error(err)
					}
					return
				}
				if err != nil {
					if err = rest.RespondWithError(w, http.StatusInternalServerError, err.Error()); err != nil {
						log.Error(err)
					}
					return
				}
				if r.Context().Value(request.User).(model.User).IsAdmin {
					tEntity := entity.(model.MediaFile)
					entity = model.AdminMediaFile(tEntity)
				}
				if err = rest.RespondWithJSON(w, 200, &entity); err != nil {
					log.Error(err)
				}
			})
			if persistable {
				r.Put("/", rest.Put(constructor))
				r.Delete("/", rest.Delete(constructor))
			}
		})
	})
}

func parseFilters(params url.Values) map[string]interface{} {
	var filterStr = params.Get("_filters")
	filters := make(map[string]interface{})
	if filterStr != "" {
		filterStr, _ = url.QueryUnescape(filterStr)
		if err := json.Unmarshal([]byte(filterStr), &filters); err != nil {
			log.Error(err)
		}
	}
	for k, v := range params {
		if strings.HasPrefix(k, "_") {
			continue
		}
		if len(v) == 1 {
			filters[k] = v[0]
		} else {
			filters[k] = v
		}
	}
	return filters
}

func parseOptions(params url.Values) rest.QueryOptions {
	start, _ := strconv.Atoi(params.Get("_start"))
	end, _ := strconv.Atoi(params.Get("_end"))

	sortField := params.Get("_sort")
	sortDir := params.Get("_order")

	return rest.QueryOptions{
		Sort:    sortField,
		Order:   strings.ToLower(sortDir),
		Offset:  start,
		Max:     int(math.Max(0, float64(end-start))),
		Filters: parseFilters(params),
	}
}

type restHandler = func(rest.RepositoryConstructor, ...rest.Logger) http.HandlerFunc

func (app *Router) addPlaylistTrackRoute(r chi.Router) {
	r.Route("/playlist/{playlistId}/tracks", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			getPlaylist(app.ds)(w, r)
		})
		r.Route("/{id}", func(r chi.Router) {
			r.Use(UrlParams)
			r.Put("/", func(w http.ResponseWriter, r *http.Request) {
				reorderItem(app.ds)(w, r)
			})
			r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
				deleteFromPlaylist(app.ds)(w, r)
			})
		})
		r.With(UrlParams).Post("/", func(w http.ResponseWriter, r *http.Request) {
			addToPlaylist(app.ds)(w, r)
		})
	})
}

// Middleware to convert Chi URL params (from Context) to query params, as expected by our REST package
func UrlParams(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := chi.RouteContext(r.Context())
		parts := make([]string, 0)
		for i, key := range ctx.URLParams.Keys {
			value := ctx.URLParams.Values[i]
			if key == "*" {
				continue
			}
			parts = append(parts, url.QueryEscape(":"+key)+"="+url.QueryEscape(value))
		}
		q := strings.Join(parts, "&")
		if r.URL.RawQuery == "" {
			r.URL.RawQuery = q
		} else {
			r.URL.RawQuery += "&" + q
		}

		next.ServeHTTP(w, r)
	})
}

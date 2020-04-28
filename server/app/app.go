package app

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/deluan/navidrome/assets"
	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/engine/auth"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/rest"
	"github.com/go-chi/chi"
	"github.com/go-chi/jwtauth"
)

type Router struct {
	ds  model.DataStore
	mux http.Handler
}

func New(ds model.DataStore) *Router {
	return &Router{ds: ds}
}

func (app *Router) Setup(path string) {
	app.mux = app.routes(path)
}

func (app *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.mux.ServeHTTP(w, r)
}

func (app *Router) routes(path string) http.Handler {
	r := chi.NewRouter()

	r.Post("/login", Login(app.ds))
	r.Post("/createAdmin", CreateAdmin(app.ds))

	r.Route("/api", func(r chi.Router) {
		r.Use(mapAuthHeader())
		r.Use(jwtauth.Verifier(auth.TokenAuth))
		r.Use(authenticator(app.ds))
		app.R(r, "/user", model.User{}, true)
		app.R(r, "/song", model.MediaFile{}, true)
		app.R(r, "/album", model.Album{}, true)
		app.R(r, "/artist", model.Artist{}, true)
		app.R(r, "/player", model.Player{}, true)
		app.R(r, "/transcoding", model.Transcoding{}, conf.Server.EnableTranscodingConfig)

		// Keepalive endpoint to be used to keep the session valid (ex: while playing songs)
		r.Get("/keepalive/*", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"response":"ok"}`)) })
	})

	// Serve UI app assets
	r.Handle("/", ServeIndex(app.ds, assets.AssetFile()))
	r.Handle("/*", http.StripPrefix(path, http.FileServer(assets.AssetFile())))

	return r
}

func (app *Router) R(r chi.Router, pathPrefix string, model interface{}, persistable bool) {
	constructor := func(ctx context.Context) rest.Repository {
		return app.ds.Resource(ctx, model)
	}
	r.Route(pathPrefix, func(r chi.Router) {
		r.Get("/", rest.GetAll(constructor))
		if persistable {
			r.Post("/", rest.Post(constructor))
		}
		r.Route("/{id:[0-9a-f\\-]+}", func(r chi.Router) {
			r.Use(UrlParams)
			r.Get("/", rest.Get(constructor))
			if persistable {
				r.Put("/", rest.Put(constructor))
				r.Delete("/", rest.Delete(constructor))
			}
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

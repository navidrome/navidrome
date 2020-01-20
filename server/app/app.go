package app

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/model"
	"github.com/cloudsonic/sonic-server/server"
	"github.com/deluan/rest"
	"github.com/go-chi/chi"
	"github.com/go-chi/jwtauth"
	"github.com/google/uuid"
)

var initialUser = model.User{
	UserName: "admin",
	Name:     "Admin",
	IsAdmin:  true,
}

type Router struct {
	ds   model.DataStore
	mux  http.Handler
	path string
}

func New(ds model.DataStore, path string) *Router {
	r := &Router{ds: ds, path: path}
	r.mux = r.routes()
	r.createDefaultUser()
	return r
}

func (app *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.mux.ServeHTTP(w, r)
}

func (app *Router) routes() http.Handler {
	r := chi.NewRouter()

	// Serve UI app assets
	server.FileServer(r, app.path, "/", http.Dir("ui/build"))

	// Basic unauthenticated ping
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"response":"pong"}`)) })

	r.Post("/login", Login(app.ds))

	r.Route("/api", func(r chi.Router) {
		// Add User resource
		r.Use(jwtauth.Verifier(TokenAuth))
		r.Use(Authenticator)
		R(r, "/user", func(ctx context.Context) rest.Repository {
			return app.ds.Resource(model.User{})
		})
	})
	return r
}

func (app *Router) createDefaultUser() {
	c, err := app.ds.User().CountAll()
	if err != nil {
		panic(fmt.Sprintf("Could not access User table: %s", err))
	}
	if c == 0 {
		id, _ := uuid.NewRandom()
		initialPassword, _ := uuid.NewRandom()
		log.Warn("Creating initial user. Please change the password!", "user", initialUser.UserName, "password", initialPassword)
		initialUser.ID = id.String()
		initialUser.Password = initialPassword.String()
		app.ds.User().Put(&initialUser)
	}
}

func R(r chi.Router, pathPrefix string, newRepository rest.RepositoryConstructor) {
	r.Route(pathPrefix, func(r chi.Router) {
		r.Get("/", rest.GetAll(newRepository))
		r.Post("/", rest.Post(newRepository))
		r.Route("/{id:[0-9a-f\\-]+}", func(r chi.Router) {
			r.Use(UrlParams)
			r.Get("/", rest.Get(newRepository))
			r.Put("/", rest.Put(newRepository))
			r.Delete("/", rest.Delete(newRepository))
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

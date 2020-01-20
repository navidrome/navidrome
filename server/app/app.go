package app

import (
	"net/http"

	"github.com/cloudsonic/sonic-server/model"
	"github.com/cloudsonic/sonic-server/server"
	"github.com/go-chi/chi"
)

type Router struct {
	ds   model.DataStore
	mux  http.Handler
	path string
}

func New(ds model.DataStore, path string) *Router {
	r := &Router{ds: ds, path: path}
	r.mux = r.routes()
	return r
}

func (app *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.mux.ServeHTTP(w, r)
}

func (app *Router) routes() http.Handler {
	r := chi.NewRouter()
	server.FileServer(r, app.path, "/", http.Dir("ui/build"))
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"response":"pong"}`)) })

	return r
}

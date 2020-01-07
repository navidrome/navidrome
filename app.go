package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi"
	chimiddleware "github.com/go-chi/chi/middleware"
	"github.com/sirupsen/logrus"
)

type App struct {
	router *chi.Mux
	logger *logrus.Logger
}

func (a *App) Initialize() {
	a.logger = logrus.New()
	a.initRoutes()
}

func (a *App) MountRouter(path string, subRouter http.Handler) {
	a.router.Group(func(r chi.Router) {
		r.Use(chimiddleware.Logger)
		r.Mount(path, subRouter)
	})
}

func (a *App) Run(addr string) {
	a.logger.Info("Listening on addr ", addr)
	a.logger.Fatal(http.ListenAndServe(addr, a.router))
}

func (a *App) initRoutes() {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Heartbeat("/ping"))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/Jamstash", 302)
	})
	workDir, _ := os.Getwd()
	filesDir := filepath.Join(workDir, "static")
	FileServer(r, "/static", http.Dir(filesDir))

	a.router = r
}

func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}

	fs := http.StripPrefix(path, http.FileServer(root))

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	}))
}

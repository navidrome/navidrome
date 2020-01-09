package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/scanner"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

type App struct {
	router   *chi.Mux
	importer *scanner.Importer
}

func (a *App) Initialize() {
	initMimeTypes()
	a.initRoutes()
	a.initImporter()
}

func (a *App) MountRouter(path string, subRouter http.Handler) {
	a.router.Group(func(r chi.Router) {
		r.Use(middleware.Logger)
		r.Mount(path, subRouter)
	})
}

func (a *App) Run(addr string) {
	log.Info("Started CloudSonic server", "address", addr)
	log.Error(http.ListenAndServe(addr, a.router))
}

func (a *App) initRoutes() {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5, "application/xml", "application/json"))
	r.Use(middleware.Heartbeat("/ping"))
	r.Use(InjectLogger)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/Jamstash", 302)
	})
	workDir, _ := os.Getwd()
	filesDir := filepath.Join(workDir, "Jamstash-master/dist")
	FileServer(r, "/Jamstash", http.Dir(filesDir))

	a.router = r
}

func (a *App) initImporter() {
	a.importer = initImporter(conf.Sonic.MusicFolder)
	go a.startPeriodicScans()
}

func (a *App) startPeriodicScans() {
	for {
		select {
		case <-time.After(5 * time.Second):
			a.importer.CheckForUpdates(false)
		}
	}
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

func InjectLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = log.NewContext(r.Context(), "requestId", ctx.Value(middleware.RequestIDKey))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

package server

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/model"
	"github.com/cloudsonic/sonic-server/scanner"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
)

const Version = "0.2"

type Server struct {
	Scanner *scanner.Scanner
	router  *chi.Mux
	ds      model.DataStore
}

func New(scanner *scanner.Scanner, ds model.DataStore) *Server {
	a := &Server{Scanner: scanner, ds: ds}
	if !conf.Sonic.DevDisableBanner {
		showBanner(Version)
	}
	initMimeTypes()
	initialSetup(ds)
	a.initRoutes()
	a.initScanner()
	return a
}

func (a *Server) MountRouter(path string, subRouter http.Handler) {
	log.Info("Mounting routes", "path", path)
	a.router.Group(func(r chi.Router) {
		r.Use(middleware.Logger)
		r.Mount(path, subRouter)
	})
}

func (a *Server) Run(addr string) {
	log.Info("CloudSonic server is accepting requests", "address", addr)
	log.Error(http.ListenAndServe(addr, a.router))
}

func (a *Server) initRoutes() {
	r := chi.NewRouter()

	r.Use(cors.Default().Handler)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5, "application/xml", "application/json", "application/javascript"))
	r.Use(middleware.Heartbeat("/ping"))
	r.Use(InjectLogger)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/app", 302)
	})

	workDir, _ := os.Getwd()
	filesDir := filepath.Join(workDir, "Jamstash-master/dist")
	FileServer(r, "/Jamstash", "/Jamstash", http.Dir(filesDir))

	a.router = r
}

func (a *Server) initScanner() {
	go func() {
		for {
			select {
			case <-time.After(5 * time.Second):
				err := a.Scanner.RescanAll(false)
				if err != nil {
					log.Error("Error scanning media folder", "folder", conf.Sonic.MusicFolder, err)
				}
			}
		}
	}()
}

func InjectLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = log.NewContext(r.Context(), "requestId", ctx.Value(middleware.RequestIDKey))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

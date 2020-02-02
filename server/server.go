package server

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/scanner"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	Scanner *scanner.Scanner
	router  *chi.Mux
	ds      model.DataStore
}

func New(scanner *scanner.Scanner, ds model.DataStore) *Server {
	a := &Server{Scanner: scanner, ds: ds}
	initMimeTypes()
	initialSetup(ds)
	a.initRoutes()
	a.initScanner()
	return a
}

func (a *Server) MountRouter(path string, subRouter http.Handler) {
	log.Info("Mounting routes", "path", path)
	a.router.Group(func(r chi.Router) {
		r.Use(RequestLogger)
		r.Mount(path, subRouter)
	})
}

func (a *Server) Run(addr string) {
	log.Info("Navidrome server is accepting requests", "address", addr)
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
	interval, err := time.ParseDuration(conf.Server.ScanInterval)
	if err != nil {
		log.Error("Invalid interval specification. Using default of 5m", "conf", conf.Server.ScanInterval, err)
		interval = 5 * time.Minute
	} else {
		log.Info("Starting scanner", "interval", interval.String())
	}
	go func() {
		time.Sleep(2 * time.Second)
		for {
			err := a.Scanner.RescanAll(false)
			if err != nil {
				log.Error("Error scanning media folder", "folder", conf.Server.MusicFolder, err)
			}
			time.Sleep(interval)
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

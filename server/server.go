package server

import (
	"net/http"
	"path"
	"time"

	"github.com/deluan/navidrome/assets"
	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/scanner"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
)

type Handler interface {
	http.Handler
	Setup(path string)
}

type Server struct {
	Scanner *scanner.Scanner
	router  *chi.Mux
	ds      model.DataStore
}

func New(scanner *scanner.Scanner, ds model.DataStore) *Server {
	a := &Server{Scanner: scanner, ds: ds}
	initialSetup(ds)
	a.initRoutes()
	a.initScanner()
	checkFfmpegInstallation()
	checkExternalCredentials()
	return a
}

func (a *Server) MountRouter(urlPath string, subRouter Handler) {
	urlPath = path.Join(conf.Server.BaseURL, urlPath)
	log.Info("Mounting routes", "path", urlPath)
	subRouter.Setup(urlPath)
	a.router.Group(func(r chi.Router) {
		r.Use(requestLogger)
		r.Mount(urlPath, subRouter)
	})
}

func (a *Server) Run(addr string) {
	log.Info("Navidrome server is accepting requests", "address", addr)
	log.Error(http.ListenAndServe(addr, a.router))
}

func (a *Server) initRoutes() {
	r := chi.NewRouter()

	r.Use(secureMiddleware())
	r.Use(cors.AllowAll().Handler)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5, "application/xml", "application/json", "application/javascript"))
	r.Use(middleware.Heartbeat("/ping"))
	r.Use(injectLogger)
	r.Use(robotsTXT(assets.AssetFile()))

	indexHtml := path.Join(conf.Server.BaseURL, consts.URLPathUI, "index.html")
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, indexHtml, 302)
	})

	a.router = r
}

func (a *Server) initScanner() {
	interval := conf.Server.ScanInterval
	if interval == 0 {
		log.Warn("Scanner is disabled", "interval", conf.Server.ScanInterval)
		return
	}
	log.Info("Starting scanner", "interval", interval.String())
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

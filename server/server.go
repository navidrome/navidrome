package server

import (
	"net/http"
	"path"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/ui"
)

type Handler interface {
	http.Handler
	Setup(path string)
}

type Server struct {
	router *chi.Mux
	ds     model.DataStore
}

func New(ds model.DataStore) *Server {
	a := &Server{ds: ds}
	initialSetup(ds)
	a.initRoutes()
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

func (a *Server) Run(addr string) error {
	log.Info("Navidrome server is accepting requests", "address", addr)
	return http.ListenAndServe(addr, a.router)
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
	r.Use(robotsTXT(ui.Assets()))

	indexHtml := path.Join(conf.Server.BaseURL, consts.URLPathUI, "index.html")
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, indexHtml, 302)
	})

	a.router = r
}

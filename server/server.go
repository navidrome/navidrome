package server

import (
	"fmt"
	"net/http"
	"path"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/ui"
)

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

func (a *Server) MountRouter(description, urlPath string, subRouter http.Handler) {
	urlPath = path.Join(conf.Server.BaseURL, urlPath)
	log.Info(fmt.Sprintf("Mounting %s routes", description), "path", urlPath)
	a.router.Group(func(r chi.Router) {
		r.Mount(urlPath, subRouter)
	})
}

func (a *Server) Run(addr string) error {
	log.Info("Navidrome server is accepting requests", "address", addr)
	return http.ListenAndServe(addr, a.router)
}

func (a *Server) initRoutes() {
	auth.Init(a.ds)

	r := chi.NewRouter()

	r.Use(secureMiddleware())
	r.Use(cors.AllowAll().Handler)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5, "application/xml", "application/json", "application/javascript"))
	r.Use(middleware.Heartbeat("/ping"))
	r.Use(injectLogger)
	r.Use(requestLogger)
	r.Use(robotsTXT(ui.Assets()))
	r.Use(authHeaderMapper)
	r.Use(jwtVerifier)

	r.Route("/auth", func(r chi.Router) {
		if conf.Server.AuthRequestLimit > 0 {
			log.Info("Login rate limit set", "requestLimit", conf.Server.AuthRequestLimit,
				"windowLength", conf.Server.AuthWindowLength)

			rateLimiter := httprate.LimitByIP(conf.Server.AuthRequestLimit, conf.Server.AuthWindowLength)
			r.With(rateLimiter).Post("/login", login(a.ds))
		} else {
			log.Warn("Login rate limit is disabled! Consider enabling it to be protected against brute-force attacks")

			r.Post("/login", login(a.ds))
		}
		r.Post("/createAdmin", createAdmin(a.ds))
	})

	// Serve UI app assets
	appRoot := path.Join(conf.Server.BaseURL, consts.URLPathUI)
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, appRoot+"/", 302)
	})
	r.Get(appRoot, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, appRoot+"/", 302)
	})
	r.Handle(appRoot+"/", serveIndex(a.ds, ui.Assets()))
	r.Handle(appRoot+"/*", http.StripPrefix(consts.URLPathUI, http.FileServer(http.FS(ui.Assets()))))

	a.router = r
}

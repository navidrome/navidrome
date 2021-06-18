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
	router  *chi.Mux
	ds      model.DataStore
	appRoot string
}

func New(ds model.DataStore) *Server {
	s := &Server{ds: ds}
	initialSetup(ds)
	s.initRoutes()
	checkFfmpegInstallation()
	checkExternalCredentials()
	return s
}

func (s *Server) MountRouter(description, urlPath string, subRouter http.Handler) {
	urlPath = path.Join(conf.Server.BaseURL, urlPath)
	log.Info(fmt.Sprintf("Mounting %s routes", description), "path", urlPath)
	s.router.Group(func(r chi.Router) {
		r.Mount(urlPath, subRouter)
	})
}

func (s *Server) Run(addr string) error {
	s.MountRouter("WebUI", consts.URLPathUI, s.frontendAssetsHandler())
	log.Info("Navidrome server is accepting requests", "address", addr)
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) initRoutes() {
	auth.Init(s.ds)

	s.appRoot = path.Join(conf.Server.BaseURL, consts.URLPathUI)

	r := chi.NewRouter()

	r.Use(secureMiddleware())
	r.Use(cors.AllowAll().Handler)
	r.Use(middleware.RequestID)
	if conf.Server.ReverseProxyWhitelist == "" {
		r.Use(middleware.RealIP)
	}
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5, "application/xml", "application/json", "application/javascript"))
	r.Use(middleware.Heartbeat("/ping"))
	r.Use(clientUniqueIdAdder)
	r.Use(loggerInjector)
	r.Use(requestLogger)
	r.Use(robotsTXT(ui.Assets()))
	r.Use(authHeaderMapper)
	r.Use(jwtVerifier)

	r.Route(path.Join(conf.Server.BaseURL, "/auth"), func(r chi.Router) {
		if conf.Server.AuthRequestLimit > 0 {
			log.Info("Login rate limit set", "requestLimit", conf.Server.AuthRequestLimit,
				"windowLength", conf.Server.AuthWindowLength)

			rateLimiter := httprate.LimitByIP(conf.Server.AuthRequestLimit, conf.Server.AuthWindowLength)
			r.With(rateLimiter).Post("/login", login(s.ds))
		} else {
			log.Warn("Login rate limit is disabled! Consider enabling it to be protected against brute-force attacks")

			r.Post("/login", login(s.ds))
		}
		r.Post("/createAdmin", createAdmin(s.ds))
	})

	// Redirect root to UI URL
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, s.appRoot+"/", http.StatusFound)
	})
	r.Get(s.appRoot, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, s.appRoot+"/", http.StatusFound)
	})

	s.router = r
}

// Serve UI app assets
func (s *Server) frontendAssetsHandler() http.Handler {
	r := chi.NewRouter()

	r.Handle("/", serveIndex(s.ds, ui.Assets()))
	r.Handle("/*", http.StripPrefix(s.appRoot, http.FileServer(http.FS(ui.Assets()))))
	return r
}

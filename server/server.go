package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
	auth.Init(s.ds)
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

func (s *Server) Run(ctx context.Context, addr string) error {
	s.MountRouter("WebUI", consts.URLPathUI, s.frontendAssetsHandler())
	server := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: consts.ServerReadHeaderTimeout,
		Handler:           s.router,
	}

	// Start HTTP server in its own goroutine, send a signal (errC) if failed to start
	errC := make(chan error)
	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Error(ctx, "Could not start server. Aborting", err)
			errC <- err
		}
	}()

	log.Info(ctx, "Navidrome server is ready!", "address", addr, "startupTime", time.Since(consts.ServerStart))

	// Wait for a signal to terminate (or an error during startup)
	select {
	case err := <-errC:
		return err
	case <-ctx.Done():
	}

	// Try to stop the HTTP server gracefully
	log.Info(ctx, "Stopping HTTP server")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		log.Error(ctx, "Unexpected error in http.Shutdown()", err)
	}
	return nil
}

func (s *Server) initRoutes() {
	s.appRoot = path.Join(conf.Server.BaseURL, consts.URLPathUI)

	r := chi.NewRouter()

	r.Use(secureMiddleware())
	r.Use(corsHandler())
	r.Use(middleware.RequestID)
	if conf.Server.ReverseProxyWhitelist == "" {
		r.Use(middleware.RealIP)
	}
	r.Use(middleware.Recoverer)
	r.Use(compressMiddleware())
	r.Use(middleware.Heartbeat("/ping"))
	r.Use(serverAddressMiddleware)
	r.Use(clientUniqueIDMiddleware)
	r.Use(loggerInjector)
	r.Use(requestLogger)
	r.Use(robotsTXT(ui.BuildAssets()))
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

	r.Handle("/", serveIndex(s.ds, ui.BuildAssets()))
	r.Handle("/*", http.StripPrefix(s.appRoot, http.FileServer(http.FS(ui.BuildAssets()))))
	return r
}

func AbsoluteURL(r *http.Request, url string, params url.Values) string {
	if strings.HasPrefix(url, "/") {
		appRoot := path.Join(r.Host, conf.Server.BaseURL, url)
		url = r.URL.Scheme + "://" + appRoot
	}
	if len(params) > 0 {
		url = url + "?" + params.Encode()
	}
	return url
}

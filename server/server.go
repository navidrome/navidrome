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
	. "github.com/navidrome/navidrome/utils/gg"
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
	urlPath = path.Join(conf.Server.BasePath, urlPath)
	log.Info(fmt.Sprintf("Mounting %s routes", description), "path", urlPath)
	s.router.Group(func(r chi.Router) {
		r.Mount(urlPath, subRouter)
	})
}

// Run starts the server with the given address, and if specified, with TLS enabled.
func (s *Server) Run(ctx context.Context, addr string, tlsCert string, tlsKey string) error {
	// Mount the router for the frontend assets
	s.MountRouter("WebUI", consts.URLPathUI, s.frontendAssetsHandler())

	// Create a new http.Server with the specified address and read header timeout
	server := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: consts.ServerReadHeaderTimeout,
		Handler:           s.router,
	}

	// Determine if TLS is enabled
	tlsEnabled := tlsCert != "" && tlsKey != ""

	// Start the server in a new goroutine and send an error signal to errC if there's an error
	errC := make(chan error)
	go func() {
		if tlsEnabled {
			// Start the HTTPS server
			log.Info("Starting server with TLS (HTTPS) enabled", "tlsCert", tlsCert, "tlsKey", tlsKey)
			if err := server.ListenAndServeTLS(tlsCert, tlsKey); !errors.Is(err, http.ErrServerClosed) {
				errC <- err
			}
		} else {
			// Start the HTTP server
			if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				errC <- err
			}
		}
	}()

	// Measure server startup time
	startupTime := time.Since(consts.ServerStart)

	// Wait a short time before checking if the server has started successfully
	time.Sleep(50 * time.Millisecond)
	select {
	case err := <-errC:
		log.Error(ctx, "Could not start server. Aborting", err)
		return fmt.Errorf("error starting server: %w", err)
	default:
		log.Info(ctx, "----> Navidrome server is ready!", "address", addr, "startupTime", startupTime, "tlsEnabled", tlsEnabled)
	}

	// Wait for a signal to terminate
	select {
	case err := <-errC:
		return fmt.Errorf("error running server: %w", err)
	case <-ctx.Done():
		// If the context is done (i.e. the server should stop), proceed to shutting down the server
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
	s.appRoot = path.Join(conf.Server.BasePath, consts.URLPathUI)

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

	r.Route(path.Join(conf.Server.BasePath, "/auth"), func(r chi.Router) {
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

	r.Handle("/", Index(s.ds, ui.BuildAssets()))
	r.Handle("/*", http.StripPrefix(s.appRoot, http.FileServer(http.FS(ui.BuildAssets()))))
	return r
}

func AbsoluteURL(r *http.Request, u string, params url.Values) string {
	buildUrl, _ := url.Parse(u)
	if strings.HasPrefix(u, "/") {
		buildUrl.Path = path.Join(conf.Server.BasePath, buildUrl.Path)
		if conf.Server.BaseHost != "" {
			buildUrl.Scheme = IfZero(conf.Server.BaseScheme, "http")
			buildUrl.Host = conf.Server.BaseHost
		} else {
			buildUrl.Scheme = r.URL.Scheme
			buildUrl.Host = r.Host
		}
	}
	if len(params) > 0 {
		buildUrl.RawQuery = params.Encode()
	}
	return buildUrl.String()
}

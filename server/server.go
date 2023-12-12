package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
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
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/ui"
	. "github.com/navidrome/navidrome/utils/gg"
)

type Server struct {
	router  chi.Router
	ds      model.DataStore
	appRoot string
	broker  events.Broker
}

func New(ds model.DataStore, broker events.Broker) *Server {
	s := &Server{ds: ds, broker: broker}
	initialSetup(ds)
	auth.Init(s.ds)
	s.initRoutes()
	s.mountAuthenticationRoutes()
	s.mountRootRedirector()
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
func (s *Server) Run(ctx context.Context, addr string, port int, tlsCert string, tlsKey string) error {
	// Mount the router for the frontend assets
	s.MountRouter("WebUI", consts.URLPathUI, s.frontendAssetsHandler())

	// Create a new http.Server with the specified read header timeout and handler
	server := &http.Server{
		ReadHeaderTimeout: consts.ServerReadHeaderTimeout,
		Handler:           s.router,
	}

	// Determine if TLS is enabled
	tlsEnabled := tlsCert != "" && tlsKey != ""

	// Create a listener based on the address type (either Unix socket or TCP)
	var listener net.Listener
	var err error
	if strings.HasPrefix(addr, "unix:") {
		socketPath := strings.TrimPrefix(addr, "unix:")
		// Remove the socket file if it already exists
		if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error removing previous unix socket file: %w", err)
		}
		listener, err = net.Listen("unix", socketPath)
		if err != nil {
			return fmt.Errorf("error creating unix socket listener: %w", err)
		}
	} else {
		addr = fmt.Sprintf("%s:%d", addr, port)
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("error creating tcp listener: %w", err)
		}
	}

	// Start the server in a new goroutine and send an error signal to errC if there's an error
	errC := make(chan error)
	go func() {
		if tlsEnabled {
			// Start the HTTPS server
			log.Info("Starting server with TLS (HTTPS) enabled", "tlsCert", tlsCert, "tlsKey", tlsKey)
			if err := server.ServeTLS(listener, tlsCert, tlsKey); !errors.Is(err, http.ErrServerClosed) {
				errC <- err
			}
		} else {
			// Start the HTTP server
			if err := server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
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
	server.SetKeepAlivesEnabled(false)
	if err := server.Shutdown(ctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		log.Error(ctx, "Unexpected error in http.Shutdown()", err)
	}
	return nil
}

func (s *Server) initRoutes() {
	s.appRoot = path.Join(conf.Server.BasePath, consts.URLPathUI)

	r := chi.NewRouter()

	middlewares := chi.Middlewares{
		secureMiddleware(),
		corsHandler(),
		middleware.RequestID,
	}
	if conf.Server.ReverseProxyWhitelist == "" {
		middlewares = append(middlewares, middleware.RealIP)
	}

	middlewares = append(middlewares,
		middleware.Recoverer,
		middleware.Heartbeat("/ping"),
		robotsTXT(ui.BuildAssets()),
		serverAddressMiddleware,
		clientUniqueIDMiddleware,
	)

	// Mount the Native API /events endpoint with all middlewares, except the compress and request logger,
	// adding the authentication middlewares
	if conf.Server.DevActivityPanel {
		r.Group(func(r chi.Router) {
			r.Use(middlewares...)
			r.Use(loggerInjector)
			r.Use(authHeaderMapper)
			r.Use(jwtVerifier)
			r.Use(Authenticator(s.ds))
			r.Use(JWTRefresher)
			r.Handle(path.Join(conf.Server.BasePath, consts.URLPathNativeAPI, "events"), s.broker)
		})
	}

	// Configure the router with the default middlewares
	r.Group(func(r chi.Router) {
		r.Use(middlewares...)
		r.Use(compressMiddleware())
		r.Use(loggerInjector)
		r.Use(requestLogger)
		r.Use(authHeaderMapper)
		r.Use(jwtVerifier)
		s.router = r
	})
}

func (s *Server) mountAuthenticationRoutes() chi.Router {
	r := s.router
	return r.Route(path.Join(conf.Server.BasePath, "/auth"), func(r chi.Router) {
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
}

// Serve UI app assets
func (s *Server) mountRootRedirector() {
	r := s.router
	// Redirect root to UI URL
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, s.appRoot+"/", http.StatusFound)
	})
	r.Get(s.appRoot, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, s.appRoot+"/", http.StatusFound)
	})
}

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
			buildUrl.Scheme = If(conf.Server.BaseScheme, "http")
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

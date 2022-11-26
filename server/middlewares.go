package server

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/cors"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/request"
	"github.com/unrolled/secure"
)

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}

		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		status := ww.Status()

		message := fmt.Sprintf("HTTP: %s %s://%s%s", r.Method, scheme, r.Host, r.RequestURI)
		logArgs := []interface{}{
			r.Context(),
			message,
			"remoteAddr", r.RemoteAddr,
			"elapsedTime", time.Since(start),
			"httpStatus", ww.Status(),
			"responseSize", ww.BytesWritten(),
		}
		if log.CurrentLevel() >= log.LevelDebug {
			logArgs = append(logArgs, "userAgent", r.UserAgent())
		}

		switch {
		case status >= 500:
			log.Error(logArgs...)
		case status >= 400:
			log.Warn(logArgs...)
		default:
			log.Debug(logArgs...)
		}
	})
}

func loggerInjector(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = log.NewContext(r.Context(), "requestId", middleware.GetReqID(ctx))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func robotsTXT(fs fs.FS) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/robots.txt") {
				r.URL.Path = "/robots.txt"
				http.FileServer(http.FS(fs)).ServeHTTP(w, r)
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}

func corsHandler() func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
		ExposedHeaders:   []string{"x-content-duration", "x-total-count", "x-nd-authorization"},
	})
}

func secureMiddleware() func(http.Handler) http.Handler {
	sec := secure.New(secure.Options{
		ContentTypeNosniff: true,
		FrameDeny:          true,
		ReferrerPolicy:     "same-origin",
		PermissionsPolicy:  "autoplay=(), camera=(), microphone=(), usb=()",
		//ContentSecurityPolicy: "script-src 'self' 'unsafe-inline'",
	})
	return sec.Handler
}

func compressMiddleware() func(http.Handler) http.Handler {
	return middleware.Compress(
		5,
		"application/xml",
		"application/json",
		"application/javascript",
		"text/html",
		"text/plain",
		"text/css",
		"text/javascript",
	)
}

func clientUniqueIdAdder(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		clientUniqueId := r.Header.Get(consts.UIClientUniqueIDHeader)
		if clientUniqueId != "" {
			c := &http.Cookie{
				Name:     consts.UIClientUniqueIDHeader,
				Value:    clientUniqueId,
				MaxAge:   consts.CookieExpiry,
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteStrictMode,
				Path:     "/",
			}
			http.SetCookie(w, c)
		} else {
			c, err := r.Cookie(consts.UIClientUniqueIDHeader)
			if !errors.Is(err, http.ErrNoCookie) {
				clientUniqueId = c.Value
			}
		}

		if clientUniqueId != "" {
			ctx = request.WithClientUniqueId(ctx, clientUniqueId)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

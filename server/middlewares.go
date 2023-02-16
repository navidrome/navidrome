package server

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/navidrome/navidrome/conf"
	. "github.com/navidrome/navidrome/utils/gg"

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

func clientUniqueIDMiddleware(next http.Handler) http.Handler {
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
				Path:     IfZero(conf.Server.BaseURL, "/"),
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

func serverAddressMiddleware(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if rScheme, rHost := serverAddress(r); rHost != "" {
			r.Host = rHost
			r.URL.Scheme = rScheme
		}
		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

var (
	xForwardedHost   = http.CanonicalHeaderKey("X-Forwarded-Host")
	xForwardedProto  = http.CanonicalHeaderKey("X-Forwarded-Scheme")
	xForwardedScheme = http.CanonicalHeaderKey("X-Forwarded-Proto")
)

func serverAddress(r *http.Request) (scheme, host string) {
	origHost := r.Host
	protocol := "http"
	if r.TLS != nil {
		protocol = "https"
	}
	xfh := r.Header.Get(xForwardedHost)
	if xfh != "" {
		i := strings.Index(xfh, ",")
		if i == -1 {
			i = len(xfh)
		}
		xfh = xfh[:i]
	}
	scheme = firstOr(
		protocol,
		r.Header.Get(xForwardedProto),
		r.Header.Get(xForwardedScheme),
		r.URL.Scheme,
	)
	host = firstOr(r.Host, xfh)
	if host != origHost {
		log.Trace(r.Context(), "Request host has changed", "origHost", origHost, "host", host, "scheme", scheme, "url", r.URL)
	}
	return scheme, host
}

func firstOr(or string, strings ...string) string {
	for _, s := range strings {
		if s != "" {
			return s
		}
	}
	return or
}

// URLParamsMiddleware convert Chi URL params (from Context) to query params, as expected by our REST package
func URLParamsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := chi.RouteContext(r.Context())
		parts := make([]string, 0)
		for i, key := range ctx.URLParams.Keys {
			value := ctx.URLParams.Values[i]
			if key == "*" {
				continue
			}
			parts = append(parts, url.QueryEscape(":"+key)+"="+url.QueryEscape(value))
		}
		q := strings.Join(parts, "&")
		if r.URL.RawQuery == "" {
			r.URL.RawQuery = q
		} else {
			r.URL.RawQuery += "&" + q
		}

		next.ServeHTTP(w, r)
	})
}

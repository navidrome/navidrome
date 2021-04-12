package server

import (
	"context"
	"fmt"
	"io/fs"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/middleware"
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

func injectLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = log.NewContext(r.Context(), "requestId", ctx.Value(middleware.RequestIDKey))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func robotsTXT(fs fs.FS) func(next http.Handler) http.Handler {
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

func secureMiddleware() func(h http.Handler) http.Handler {
	sec := secure.New(secure.Options{
		ContentTypeNosniff: true,
		FrameDeny:          true,
		ReferrerPolicy:     "same-origin",
		FeaturePolicy:      "autoplay 'none'; camera: 'none'; display-capture 'none'; microphone: 'none'; usb: 'none'",
		//ContentSecurityPolicy: "script-src 'self' 'unsafe-inline'",
	})
	return sec.Handler
}

func randomSeedMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ctx context.Context
		switch r.URL.Path {
		case "/app/api/album":
			if sort, ok := r.URL.Query()["_sort"]; ok && sort[0] == "random" {
				seed := int64(0)
				c, err := r.Cookie("albumRandom")
				if err == nil {
					seed, err = strconv.ParseInt(c.Value, 16, 64)
				}
				if _, ok := r.URL.Query()["_shuffle"]; ok || err != nil {
					seed = rand.NewSource(time.Now().UnixNano()).Int63()
					c := http.Cookie{
						Name:  "albumRandom",
						Value: strconv.FormatInt(seed, 16),
						// TODO: Get rid of expire when _shuffle works
						Expires: time.Now().Add(time.Minute * 5),
					}
					http.SetCookie(w, &c)
				}
				ctx = context.WithValue(r.Context(), request.AlbumRandomSeed, seed)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

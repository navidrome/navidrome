package subsonic

import (
	"cmp"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	ua "github.com/mileusna/useragent"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

// mergeFormIntoQuery parses form data (both URL query params and POST body)
// and writes all values back into r.URL.RawQuery. This is needed because
// some Subsonic clients send parameters as form fields instead of query params.
// This support the OpenSubsonic `formPost` extension
func mergeFormIntoQuery(r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	var parts []string
	for key, values := range r.Form {
		for _, v := range values {
			parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(v))
		}
	}
	r.URL.RawQuery = strings.Join(parts, "&")
	return nil
}

func postFormToQueryParams(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := mergeFormIntoQuery(r); err != nil {
			sendError(w, r, newError(responses.ErrorGeneric, err.Error()))
		}
		next.ServeHTTP(w, r)
	})
}

func fromInternalOrProxyAuth(r *http.Request) (string, bool) {
	username := server.InternalAuth(r)

	// If the username comes from internal auth, do not also do reverse proxy auth, as
	// the request will have no reverse proxy IP
	if username != "" {
		return username, true
	}

	return server.UsernameFromExtAuthHeader(r), false
}

func checkRequiredParameters(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requiredParameters []string

		username, _ := fromInternalOrProxyAuth(r)
		if username != "" {
			requiredParameters = []string{"v", "c"}
		} else {
			requiredParameters = []string{"u", "v", "c"}
		}

		p := req.Params(r)
		for _, param := range requiredParameters {
			if _, err := p.String(param); err != nil {
				log.Warn(r, err)
				sendError(w, r, err)
				return
			}
		}

		if username == "" {
			username, _ = p.String("u")
		}
		client, _ := p.String("c")
		version, _ := p.String("v")

		ctx := r.Context()
		ctx = request.WithUsername(ctx, username)
		ctx = request.WithClient(ctx, client)
		ctx = request.WithVersion(ctx, version)
		log.Debug(ctx, "API: New request "+r.URL.Path, "username", username, "client", client, "version", version)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authenticateRequest validates the authentication credentials in an HTTP request and returns
// the authenticated user. It supports internal auth, reverse proxy auth, and Subsonic classic
// auth (username + password/token/salt/jwt query params).
//
// Callers should handle specific error types as needed:
//   - context.Canceled: request was canceled during authentication
//   - model.ErrNotFound: username not found in database
//   - model.ErrInvalidAuth: invalid credentials (wrong password, token, etc.)
func authenticateRequest(ds model.DataStore, r *http.Request) (*model.User, error) {
	ctx := r.Context()

	// Check internal auth or reverse proxy auth first
	username, _ := fromInternalOrProxyAuth(r)
	if username != "" {
		return ds.User(ctx).FindByUsername(username)
	}

	// Fall back to Subsonic classic auth (query params)
	p := req.Params(r)
	username, _ = p.String("u")
	if username == "" {
		return nil, model.ErrInvalidAuth
	}

	pass, _ := p.String("p")
	token, _ := p.String("t")
	salt, _ := p.String("s")
	jwt, _ := p.String("jwt")

	usr, err := ds.User(ctx).FindByUsernameWithPassword(username)
	if err != nil {
		return nil, err
	}

	if err := validateCredentials(usr, pass, token, salt, jwt); err != nil {
		return nil, err
	}

	return usr, nil
}

func authenticate(ds model.DataStore) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			usr, err := authenticateRequest(ds, r)
			if err != nil {
				username, _ := request.UsernameFrom(ctx)
				switch {
				case errors.Is(err, context.Canceled):
					log.Debug(ctx, "API: Request canceled when authenticating", "username", username, "remoteAddr", r.RemoteAddr, err)
					return
				case errors.Is(err, model.ErrNotFound), errors.Is(err, model.ErrInvalidAuth):
					log.Warn(ctx, "API: Invalid login", "username", username, "remoteAddr", r.RemoteAddr, err)
				default:
					log.Error(ctx, "API: Error authenticating", "username", username, "remoteAddr", r.RemoteAddr, err)
				}
				sendError(w, r, newError(responses.ErrorAuthenticationFail))
				return
			}

			ctx = request.WithUser(ctx, *usr)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ValidateAuth validates Subsonic authentication from an HTTP request and returns the authenticated user.
// Unlike the authenticate middleware, this function does not write any HTTP response, making it suitable
// for use by external consumers (e.g., plugin endpoints) that need Subsonic auth but want to handle
// errors themselves.
func ValidateAuth(ds model.DataStore, r *http.Request) (*model.User, error) {
	// Parse form data into query params (same as postFormToQueryParams middleware,
	// which is not in the call chain when ValidateAuth is used directly)
	if err := mergeFormIntoQuery(r); err != nil {
		return nil, fmt.Errorf("parsing form: %w", err)
	}
	return authenticateRequest(ds, r)
}

func validateCredentials(user *model.User, pass, token, salt, jwt string) error {
	valid := false

	switch {
	case jwt != "":
		claims, err := auth.Validate(jwt)
		valid = err == nil && claims["sub"] == user.UserName
	case pass != "":
		if strings.HasPrefix(pass, "enc:") {
			if dec, err := hex.DecodeString(pass[4:]); err == nil {
				pass = string(dec)
			}
		}
		valid = pass == user.Password
	case token != "":
		t := fmt.Sprintf("%x", md5.Sum([]byte(user.Password+salt)))
		valid = t == token
	}

	if !valid {
		return model.ErrInvalidAuth
	}
	return nil
}

func getPlayer(players core.Players) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userName, _ := request.UsernameFrom(ctx)
			client, _ := request.ClientFrom(ctx)
			playerId := playerIDFromCookie(r, userName)
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			userAgent := canonicalUserAgent(r)
			player, trc, err := players.Register(ctx, playerId, client, userAgent, ip)
			if err != nil {
				log.Error(ctx, "Could not register player", "username", userName, "client", client, err)
			} else {
				ctx = request.WithPlayer(ctx, *player)
				if trc != nil {
					ctx = request.WithTranscoding(ctx, *trc)
				}
				r = r.WithContext(ctx)

				cookie := &http.Cookie{
					Name:     playerIDCookieName(userName),
					Value:    player.ID,
					MaxAge:   consts.CookieExpiry,
					HttpOnly: true,
					SameSite: http.SameSiteStrictMode,
					Path:     cmp.Or(conf.Server.BasePath, "/"),
				}
				http.SetCookie(w, cookie)
			}

			next.ServeHTTP(w, r)
		})
	}
}

func canonicalUserAgent(r *http.Request) string {
	u := ua.Parse(r.Header.Get("user-agent"))
	userAgent := u.Name
	if u.OS != "" {
		userAgent = userAgent + "/" + u.OS
	}
	return userAgent
}

func playerIDFromCookie(r *http.Request, userName string) string {
	cookieName := playerIDCookieName(userName)
	var playerId string
	if c, err := r.Cookie(cookieName); err == nil {
		playerId = c.Value
		log.Trace(r, "playerId found in cookies", "playerId", playerId)
	}
	return playerId
}

func playerIDCookieName(userName string) string {
	cookieName := fmt.Sprintf("nd-player-%x", userName)
	return cookieName
}

const subsonicErrorPointer = "subsonicErrorPointer"

func recordStats(metrics metrics.Metrics) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			status := int32(-1)
			contextWithStatus := context.WithValue(r.Context(), subsonicErrorPointer, &status)

			start := time.Now()
			defer func() {
				elapsed := time.Since(start).Milliseconds()

				// We want to get the client name (even if not present for certain endpoints)
				p := req.Params(r)
				client, _ := p.String("c")

				// If there is no Subsonic status (e.g., HTTP 501 not implemented), fallback to HTTP
				if status == -1 {
					status = int32(ww.Status())
				}

				shortPath := strings.Replace(r.URL.Path, ".view", "", 1)

				metrics.RecordRequest(r.Context(), shortPath, r.Method, client, status, elapsed)
			}()

			next.ServeHTTP(ww, r.WithContext(contextWithStatus))
		}
		return http.HandlerFunc(fn)
	}
}

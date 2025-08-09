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
	. "github.com/navidrome/navidrome/utils/gg"
	"github.com/navidrome/navidrome/utils/req"
)

func postFormToQueryParams(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			sendError(w, r, newError(responses.ErrorGeneric, err.Error()))
		}
		var parts []string
		for key, values := range r.Form {
			for _, v := range values {
				parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(v))
			}
		}
		r.URL.RawQuery = strings.Join(parts, "&")

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

	return server.UsernameFromReverseProxyHeader(r), false
}

func checkRequiredParameters(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requiredParameters []string

		p := req.Params(r)
		username, _ := fromInternalOrProxyAuth(r)
		if username != "" {
			requiredParameters = []string{"v", "c"}
		} else if apiKey, _ := p.String("apiKey"); apiKey != "" {
			requiredParameters = []string{"v", "c"}
		} else {
			requiredParameters = []string{"u", "v", "c"}
		}

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

func authenticate(ds model.DataStore) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			var usr *model.User
			var err error

			username, isInternalAuth := fromInternalOrProxyAuth(r)
			if username != "" {
				authType := If(isInternalAuth, "internal", "reverse-proxy")
				usr, err = ds.User(ctx).FindByUsername(username)
				if errors.Is(err, context.Canceled) {
					log.Debug(ctx, "API: Request canceled when authenticating", "auth", authType, "username", username, "remoteAddr", r.RemoteAddr, err)
					return
				}
				if errors.Is(err, model.ErrNotFound) {
					log.Warn(ctx, "API: Invalid login", "auth", authType, "username", username, "remoteAddr", r.RemoteAddr, err)
				} else if err != nil {
					log.Error(ctx, "API: Error authenticating username", "auth", authType, "username", username, "remoteAddr", r.RemoteAddr, err)
				}
			} else {
				p := req.Params(r)
				username, _ := p.String("u")
				pass, _ := p.String("p")
				token, _ := p.String("t")
				salt, _ := p.String("s")
				jwt, _ := p.String("jwt")
				apiKey, _ := p.String("apiKey")

				// When an API key is provided, username should not be provided
				if apiKey != "" && username != "" {
					log.Warn(ctx, "API: Invalid login - username provided with API key", "auth", "subsonic", "remoteAddr", r.RemoteAddr)
					sendError(w, r, newError(responses.ErrorMultipleAuthMechanismsProvided))
					return
				}

				// Check for conflicting authentication mechanisms
				authMechanismsCount := 0
				if apiKey != "" {
					authMechanismsCount++
				}
				if pass != "" {
					authMechanismsCount++
				}
				if token != "" && salt != "" {
					authMechanismsCount++
				}
				if jwt != "" {
					authMechanismsCount++
				}
				if authMechanismsCount > 1 {
					log.Warn(ctx, "API: Invalid login - multiple authentication mechanisms", "auth", "subsonic", "remoteAddr", r.RemoteAddr)
					sendError(w, r, newError(responses.ErrorMultipleAuthMechanismsProvided))
					return
				}

				if apiKey != "" {
					var player *model.Player
					player, err = ds.Player(ctx).FindByAPIKey(apiKey)
					if errors.Is(err, context.Canceled) {
						log.Debug(ctx, "API: Request canceled when authenticating", "auth", "subsonic", "remoteAddr", r.RemoteAddr, err)
						return
					}
					switch {
					case errors.Is(err, model.ErrNotFound):
						log.Warn(ctx, "API: Invalid login - API key not found", "auth", "subsonic", "apikey", apiKey, "remoteAddr", r.RemoteAddr, err)
					case err != nil:
						log.Error(ctx, "API: Error authenticating with API key", "auth", "subsonic", "apikey", apiKey, "remoteAddr", r.RemoteAddr, err)
					default:
						usr, err = ds.User(ctx).Get(player.UserId)
						if err != nil {
							log.Error(ctx, "API: Error retrieving user from API key", "auth", "subsonic", "apikey", apiKey, "remoteAddr", r.RemoteAddr, err)
						}
					}
				} else {
					usr, err = ds.User(ctx).FindByUsernameWithPassword(username)
					if errors.Is(err, context.Canceled) {
						log.Debug(ctx, "API: Request canceled when authenticating", "auth", "subsonic", "username", username, "remoteAddr", r.RemoteAddr, err)
						return
					}
					switch {
					case errors.Is(err, model.ErrNotFound):
						log.Warn(ctx, "API: Invalid login", "auth", "subsonic", "username", username, "remoteAddr", r.RemoteAddr, err)
					case err != nil:
						log.Error(ctx, "API: Error authenticating username", "auth", "subsonic", "username", username, "remoteAddr", r.RemoteAddr, err)
					default:
						err = validateCredentials(usr, pass, token, salt, jwt)
						if err != nil {
							log.Warn(ctx, "API: Invalid login", "auth", "subsonic", "username", username, "remoteAddr", r.RemoteAddr, err)
						}
					}
				}
			}

			if err != nil {
				sendError(w, r, newError(responses.ErrorAuthenticationFail))
				return
			}

			ctx = request.WithUser(ctx, *usr)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
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

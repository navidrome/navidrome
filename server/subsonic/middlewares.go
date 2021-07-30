package subsonic

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	ua "github.com/mileusna/useragent"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils"
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

func checkRequiredParameters(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requiredParameters := []string{"u", "v", "c"}

		for _, p := range requiredParameters {
			if utils.ParamString(r, p) == "" {
				msg := fmt.Sprintf(`Missing required parameter "%s"`, p)
				log.Warn(r, msg)
				sendError(w, r, newError(responses.ErrorMissingParameter, msg))
				return
			}
		}

		username := utils.ParamString(r, "u")
		client := utils.ParamString(r, "c")
		version := utils.ParamString(r, "v")
		ctx := r.Context()
		ctx = request.WithUsername(ctx, username)
		ctx = request.WithClient(ctx, client)
		ctx = request.WithVersion(ctx, version)
		log.Debug(ctx, "API: New request "+r.URL.Path, "username", username, "client", client, "version", version)

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func authenticate(ds model.DataStore) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			username, pass, token, salt, jwt := extractAuthParameters(r)

			usr, err := validateUser(ctx, ds, username, pass, token, salt, jwt)
			if err == model.ErrInvalidAuth {
				log.Warn(ctx, "Invalid login", "username", username, err)
			} else if err != nil {
				log.Error(ctx, "Error authenticating username", "username", username, err)
			}

			if err != nil {
				sendError(w, r, newError(responses.ErrorAuthenticationFail))
				return
			}

			// TODO: Find a way to update LastAccessAt without causing too much retention in the DB
			//go func() {
			//	err := ds.User(ctx).UpdateLastAccessAt(usr.ID)
			//	if err != nil {
			//		log.Error(ctx, "Could not update user's lastAccessAt", "user", usr.UserId)
			//	}
			//}()

			ctx = request.WithUser(ctx, *usr)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

func extractAuthParameters(r *http.Request) (string, string, string, string, string) {
	username := utils.ParamString(r, "u")
	pass := utils.ParamString(r, "p")
	token := utils.ParamString(r, "t")
	salt := utils.ParamString(r, "s")

	jwt := ""
	if c, err := r.Cookie(consts.UIAuthorizationHeader); err == nil {
		jwt = c.Value
	}

	return username, pass, token, salt, jwt
}

func validateUser(ctx context.Context, ds model.DataStore, username, pass, token, salt, jwt string) (*model.User, error) {
	user, err := ds.User(ctx).FindByUsernameWithPassword(username)
	if err == model.ErrNotFound {
		return nil, model.ErrInvalidAuth
	}
	if err != nil {
		return nil, err
	}

	switch {
	case jwt != "":
		log.Debug(ctx, "AUTH: Using Jwt to validate User", "user", user.UserName)
		return validateJwt(user, jwt)
	case pass != "":
		log.Debug(ctx, "AUTH: Using Password to validate User", "user", user.UserName)
		return validatePass(user, pass)
	case token != "":
		log.Debug(ctx, "AUTH: Using SubsonicToken to validate User", "user", user.UserName)
		return validateToken(user, token, salt)
	default:
		log.Warn(ctx, "AUTH: Received no valid authentication method", "user", user.UserName)
		return nil, model.ErrInvalidAuth
	}
}

func validateJwt(user *model.User, jwt string) (*model.User, error) {
	claims, err := auth.Validate(jwt)

	if err == nil && claims["sub"] == user.UserName {
		return user, nil
	}

	return nil, model.ErrInvalidAuth
}

func validatePass(user *model.User, pass string) (*model.User, error) {
	if pass == user.Password {
		return user, nil
	}

	if strings.HasPrefix(pass, "enc:") {
		if dec, err := hex.DecodeString(pass[4:]); err == nil {
			if string(dec) == user.Password {
				return user, nil
			}
		}

	}

	return nil, model.ErrInvalidAuth
}

func validateToken(user *model.User, token string, salt string) (*model.User, error) {
	t := fmt.Sprintf("%x", md5.Sum([]byte(user.Password+salt)))

	if t == token {
		return user, nil
	}

	return nil, model.ErrInvalidAuth
}

func getPlayer(players core.Players) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userName, _ := request.UsernameFrom(ctx)
			client, _ := request.ClientFrom(ctx)
			playerId := playerIDFromCookie(r, userName)
			ip, _, _ := net.SplitHostPort(realIP(r))
			userAgent := canonicalUserAgent(r)
			player, trc, err := players.Register(ctx, playerId, client, userAgent, ip)
			if err != nil {
				log.Error("Could not register player", "username", userName, "client", client, err)
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
					SameSite: http.SameSiteStrictMode,
					HttpOnly: true,
					Path:     "/",
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

func realIP(r *http.Request) string {
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	} else if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		i := strings.Index(xff, ", ")
		if i == -1 {
			i = len(xff)
		}
		return xff[:i]
	}
	return r.RemoteAddr
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

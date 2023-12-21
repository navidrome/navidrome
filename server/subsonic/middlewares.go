package subsonic

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	ua "github.com/mileusna/useragent"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
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

func checkRequiredParameters(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requiredParameters := []string{"u", "v", "c"}
		p := req.Params(r)
		for _, param := range requiredParameters {
			if _, err := p.String(param); err != nil {
				log.Warn(r, err)
				sendError(w, r, err)
				return
			}
		}

		username, _ := p.String("u")
		client, _ := p.String("c")
		version, _ := p.String("v")
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
			p := req.Params(r)
			username, _ := p.String("u")

			pass, _ := p.String("p")
			token, _ := p.String("t")
			salt, _ := p.String("s")
			jwt, _ := p.String("jwt")

			usr, err := validateUser(ctx, ds, username, pass, token, salt, jwt)
			if errors.Is(err, model.ErrInvalidAuth) {
				log.Warn(ctx, "API: Invalid login", "username", username, "remoteAddr", r.RemoteAddr, err)
			} else if err != nil {
				log.Error(ctx, "API: Error authenticating username", "username", username, "remoteAddr", r.RemoteAddr, err)
			}

			if err != nil {
				sendError(w, r, newError(responses.ErrorAuthenticationFail))
				return
			}

			// TODO: Find a way to update LastAccessAt without causing too much retention in the DB
			//go func() {
			//	err := ds.User(ctx).UpdateLastAccessAt(usr.ID)
			//	if err != nil {
			//		log.Error(ctx, "Could not update user's lastAccessAt", "user", usr.UserName)
			//	}
			//}()

			ctx = log.NewContext(r.Context(), "username", username)
			ctx = request.WithUser(ctx, *usr)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

func validateUser(ctx context.Context, ds model.DataStore, username, pass, token, salt, jwt string) (*model.User, error) {
	user, err := ds.User(ctx).FindByUsernameWithPassword(username)
	if errors.Is(err, model.ErrNotFound) {
		return nil, model.ErrInvalidAuth
	}
	if err != nil {
		return nil, err
	}
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
		return nil, model.ErrInvalidAuth
	}
	return user, nil
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
				log.Error(r.Context(), "Could not register player", "username", userName, "client", client, err)
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
					Path:     If(conf.Server.BasePath, "/"),
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

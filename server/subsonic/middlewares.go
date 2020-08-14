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

	"github.com/deluan/navidrome/core/auth"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/model/request"
	"github.com/deluan/navidrome/server/subsonic/engine"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
)

const (
	cookieExpiry = 365 * 24 * 3600 // One year
)

func postFormToQueryParams(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			SendError(w, r, newError(responses.ErrorGeneric, err.Error()))
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
				SendError(w, r, newError(responses.ErrorMissingParameter, msg))
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
			username := utils.ParamString(r, "u")

			pass := utils.ParamString(r, "p")
			token := utils.ParamString(r, "t")
			salt := utils.ParamString(r, "s")
			jwt := utils.ParamString(r, "jwt")

			usr, err := validateUser(ctx, ds, username, pass, token, salt, jwt)
			if err == model.ErrInvalidAuth {
				log.Warn(ctx, "Invalid login", "username", username, err)
			} else if err != nil {
				log.Error(ctx, "Error authenticating username", "username", username, err)
			}

			if err != nil {
				SendError(w, r, newError(responses.ErrorAuthenticationFail))
				return
			}

			// TODO: Find a way to update LastAccessAt without causing too much retention in the DB
			//go func() {
			//	err := ds.User(ctx).UpdateLastAccessAt(usr.ID)
			//	if err != nil {
			//		log.Error(ctx, "Could not update user's lastAccessAt", "user", usr.UserName)
			//	}
			//}()

			ctx = request.WithUser(ctx, *usr)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

func validateUser(ctx context.Context, ds model.DataStore, username, pass, token, salt, jwt string) (*model.User, error) {
	user, err := ds.User(ctx).FindByUsername(username)
	if err == model.ErrNotFound {
		return nil, model.ErrInvalidAuth
	}
	if err != nil {
		return nil, err
	}
	valid := false

	switch {
	case jwt != "":
		claims, err := auth.Validate(jwt)
		valid = err == nil && claims["sub"] == username
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

func getPlayer(players engine.Players) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userName, _ := request.UsernameFrom(ctx)
			client, _ := request.ClientFrom(ctx)
			playerId := playerIDFromCookie(r, userName)
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			player, trc, err := players.Register(ctx, playerId, client, r.Header.Get("user-agent"), ip)
			if err != nil {
				log.Error("Could not register player", "username", userName, "client", client)
			} else {
				ctx = request.WithPlayer(ctx, *player)
				if trc != nil {
					ctx = request.WithTranscoding(ctx, *trc)
				}
				r = r.WithContext(ctx)

				cookie := &http.Cookie{
					Name:     playerIDCookieName(userName),
					Value:    player.ID,
					MaxAge:   cookieExpiry,
					HttpOnly: true,
					Path:     "/",
				}
				http.SetCookie(w, cookie)
			}

			next.ServeHTTP(w, r)
		})
	}
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

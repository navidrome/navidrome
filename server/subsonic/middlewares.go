package subsonic

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/deluan/navidrome/engine"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
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
			SendError(w, r, NewError(responses.ErrorGeneric, err.Error()))
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
				SendError(w, r, NewError(responses.ErrorMissingParameter, msg))
				return
			}
		}

		user := utils.ParamString(r, "u")
		client := utils.ParamString(r, "c")
		version := utils.ParamString(r, "v")
		ctx := r.Context()
		ctx = context.WithValue(ctx, "username", user)
		ctx = context.WithValue(ctx, "client", client)
		ctx = context.WithValue(ctx, "version", version)
		log.Info(ctx, "API: New request "+r.URL.Path, "username", user, "client", client, "version", version)

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func authenticate(users engine.Users) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username := utils.ParamString(r, "u")
			pass := utils.ParamString(r, "p")
			token := utils.ParamString(r, "t")
			salt := utils.ParamString(r, "s")
			jwt := utils.ParamString(r, "jwt")

			usr, err := users.Authenticate(r.Context(), username, pass, token, salt, jwt)
			if err == model.ErrInvalidAuth {
				log.Warn(r, "Invalid login", "username", username, err)
			} else if err != nil {
				log.Error(r, "Error authenticating username", "username", username, err)
			}

			if err != nil {
				log.Warn(r, "Invalid login", "username", username)
				SendError(w, r, NewError(responses.ErrorAuthenticationFail))
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, "user", *usr)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

func getPlayer(players engine.Players) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userName := ctx.Value("username").(string)
			client := ctx.Value("client").(string)
			playerId := playerIDFromCookie(r, userName)
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			player, trc, err := players.Register(ctx, playerId, client, r.Header.Get("user-agent"), ip)
			if err != nil {
				log.Error("Could not register player", "userName", userName, "client", client)
			} else {
				ctx = context.WithValue(ctx, "player", *player)
				if trc != nil {
					ctx = context.WithValue(ctx, "transcoding", *trc)
				}
				r = r.WithContext(ctx)
			}

			cookie := &http.Cookie{
				Name:     playerIDCookieName(userName),
				Value:    player.ID,
				MaxAge:   cookieExpiry,
				HttpOnly: true,
				Path:     "/",
			}
			http.SetCookie(w, cookie)
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

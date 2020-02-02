package subsonic

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/deluan/navidrome/engine"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/server/subsonic/responses"
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
			if ParamString(r, p) == "" {
				msg := fmt.Sprintf(`Missing required parameter "%s"`, p)
				log.Warn(r, msg)
				SendError(w, r, NewError(responses.ErrorMissingParameter, msg))
				return
			}
		}

		if ParamString(r, "p") == "" && (ParamString(r, "s") == "" || ParamString(r, "t") == "") {
			log.Warn(r, "Missing authentication information")
		}

		user := ParamString(r, "u")
		client := ParamString(r, "c")
		version := ParamString(r, "v")
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
			username := ParamString(r, "u")
			pass := ParamString(r, "p")
			token := ParamString(r, "t")
			salt := ParamString(r, "s")

			usr, err := users.Authenticate(r.Context(), username, pass, token, salt)
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
			ctx = context.WithValue(ctx, "user", usr)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

func requiredParams(params ...string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, p := range params {
				_, err := RequiredParamString(r, p, fmt.Sprintf("%s parameter is required", p))
				if err != nil {
					SendError(w, r, err)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

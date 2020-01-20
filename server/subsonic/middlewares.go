package subsonic

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/model"
	"github.com/cloudsonic/sonic-server/server/subsonic/responses"
)

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
		log.Info(ctx, "New Subsonic API request", "username", user, "client", client, "version", version, "path", r.URL.Path)

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

			usr, err := users.Authenticate(username, pass, token, salt)
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

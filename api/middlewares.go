package api

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudsonic/sonic-server/api/responses"
	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/log"
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
		ctx = context.WithValue(ctx, "user", user)
		ctx = context.WithValue(ctx, "client", client)
		ctx = context.WithValue(ctx, "version", version)
		log.Info(ctx, "New Subsonic API request", "user", user, "client", client, "version", version, "path", r.URL.Path)

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		password := conf.Sonic.Password
		user := ParamString(r, "u")
		pass := ParamString(r, "p")
		salt := ParamString(r, "s")
		token := ParamString(r, "t")
		valid := false

		switch {
		case pass != "":
			if strings.HasPrefix(pass, "enc:") {
				if dec, err := hex.DecodeString(pass[4:]); err == nil {
					pass = string(dec)
				}
			}
			valid = pass == password
		case token != "":
			t := fmt.Sprintf("%x", md5.Sum([]byte(password+salt)))
			valid = t == token
		}

		if user != conf.Sonic.User || !valid {
			log.Warn(r, "Invalid login", "user", user)
			SendError(w, r, NewError(responses.ErrorAuthenticationFail))
			return
		}

		next.ServeHTTP(w, r)
	})
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

package api

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/astaxie/beego"
	"github.com/cloudsonic/sonic-server/api/responses"
	"github.com/cloudsonic/sonic-server/conf"
)

func checkRequiredParameters(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requiredParameters := []string{"u", "v", "c"}

		for _, p := range requiredParameters {
			if ParamString(r, p) == "" {
				msg := fmt.Sprintf(`Missing required parameter "%s"`, p)
				beego.Warn(msg)
				SendError(w, r, NewError(responses.ErrorMissingParameter, msg))
				return
			}
		}

		if ParamString(r, "p") == "" && (ParamString(r, "s") == "" || ParamString(r, "t") == "") {
			beego.Warn("Missing authentication information")
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, "user", ParamString(r, "u"))
		ctx = context.WithValue(ctx, "client", ParamString(r, "c"))
		ctx = context.WithValue(ctx, "version", ParamString(r, "v"))
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
				e := strings.TrimPrefix(pass, "enc:")
				if dec, err := hex.DecodeString(e); err == nil {
					pass = string(dec)
				}
			}
			valid = pass == password
		case token != "":
			t := fmt.Sprintf("%x", md5.Sum([]byte(password+salt)))
			valid = t == token
		}

		if user != conf.Sonic.User || !valid {
			beego.Warn(fmt.Sprintf(`Invalid login for user "%s"`, user))
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

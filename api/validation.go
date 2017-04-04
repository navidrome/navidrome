package api

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/astaxie/beego"
	"github.com/cloudsonic/sonic-server/api/responses"
	"github.com/cloudsonic/sonic-server/conf"
)

type ControllerInterface interface {
	GetString(key string, def ...string) string
	CustomAbort(status int, body string)
	SendError(errorCode int, message ...interface{})
}

func Validate(controller BaseAPIController) {
	addNewContext(controller)
	if !conf.Sonic.DisableValidation {
		checkParameters(controller)
		authenticate(controller)
		// TODO Validate version
	}
}

func addNewContext(c BaseAPIController) {
	ctx := context.Background()

	id := c.Ctx.Input.GetData("requestId")
	ctx = context.WithValue(ctx, "requestId", id)
	c.Ctx.Input.SetData("context", ctx)
}

func checkParameters(c BaseAPIController) {
	requiredParameters := []string{"u", "v", "c"}

	for _, p := range requiredParameters {
		if c.GetString(p) == "" {
			logWarn(c, fmt.Sprintf(`Missing required parameter "%s"`, p))
			abortRequest(c, responses.ErrorMissingParameter)
		}
	}

	if c.GetString("p") == "" && (c.GetString("s") == "" || c.GetString("t") == "") {
		logWarn(c, "Missing authentication information")
	}
	ctx := c.Ctx.Input.GetData("context").(context.Context)
	ctx = context.WithValue(ctx, "user", c.GetString("u"))
	ctx = context.WithValue(ctx, "client", c.GetString("c"))
	ctx = context.WithValue(ctx, "version", c.GetString("v"))
	c.Ctx.Input.SetData("context", ctx)
}

func authenticate(c BaseAPIController) {
	password := conf.Sonic.Password
	user := c.GetString("u")
	pass := c.GetString("p")
	salt := c.GetString("s")
	token := c.GetString("t")
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
		logWarn(c, fmt.Sprintf(`Invalid login for user "%s"`, user))
		abortRequest(c, responses.ErrorAuthenticationFail)
	}
}

func abortRequest(c BaseAPIController, code int) {
	c.SendError(code)
}

func logWarn(c BaseAPIController, msg string) {
	beego.Warn(fmt.Sprintf("%s?%s: %s", c.Ctx.Request.URL.Path, c.Ctx.Request.URL.RawQuery, msg))
}

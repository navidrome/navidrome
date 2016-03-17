package api

import (
	"encoding/hex"
	"strings"

	"fmt"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
)

type ControllerInterface interface {
	GetString(key string, def ...string) string
	CustomAbort(status int, body string)
	SendError(errorCode int, message ...interface{})
}

func Validate(controller BaseAPIController) {
	if beego.AppConfig.String("disableValidation") != "true" {
		checkParameters(controller)
		authenticate(controller)
		// TODO Validate version
	}
}

func checkParameters(c BaseAPIController) {
	requiredParameters := []string{"u", "p", "v", "c"}

	for _, p := range requiredParameters {
		if c.GetString(p) == "" {
			logWarn(c, fmt.Sprintf(`Missing required parameter "%s"`, p))
			abortRequest(c, responses.ERROR_MISSING_PARAMETER)
		}
		c.Data[p] = c.GetString(p)
	}
}

func authenticate(c BaseAPIController) {
	user := c.GetString("u")
	pass := c.GetString("p")
	if strings.HasPrefix(pass, "enc:") {
		e := strings.TrimPrefix(pass, "enc:")
		if dec, err := hex.DecodeString(e); err == nil {
			pass = string(dec)
		}
	}
	if user != beego.AppConfig.String("user") || pass != beego.AppConfig.String("password") {
		logWarn(c, fmt.Sprintf(`Invalid login for user "%s"`, user))
		abortRequest(c, responses.ERROR_AUTHENTICATION_FAIL)
	}
}

func abortRequest(c BaseAPIController, code int) {
	c.SendError(code)
}

func logWarn(c BaseAPIController, msg string) {
	beego.Warn(fmt.Sprintf("%s?%s: %s", c.Ctx.Request.URL.Path, c.Ctx.Request.URL.RawQuery, msg))
}

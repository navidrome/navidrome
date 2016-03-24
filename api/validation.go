package api

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"

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
}

func authenticate(c BaseAPIController) {
	password := beego.AppConfig.String("password")
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
		valid = (pass == password)
	case token != "":
		t := fmt.Sprintf("%x", md5.Sum([]byte(password+salt)))
		valid = (t == token)
	}

	if user != beego.AppConfig.String("user") || !valid {
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

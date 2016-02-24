package controllers

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/controllers/responses"
)

type ControllerInterface interface {
	GetString(key string, def ...string) string
	CustomAbort(status int, body string)
}

func validate(controller ControllerInterface) {
	if beego.AppConfig.String("disableValidation") != "true" {
		checkParameters(controller)
		authenticate(controller)
	}
}

func checkParameters(c ControllerInterface) {
	requiredParameters := []string {"u", "p", "v", "c",}

	for _,p := range requiredParameters {
		if c.GetString(p) == "" {
			cancel(c, responses.ERROR_MISSING_PARAMETER)
		}
	}
}

func authenticate(c ControllerInterface) {
	user := c.GetString("u")
	pass := c.GetString("p")
	if (user != beego.AppConfig.String("user") || pass != beego.AppConfig.String("password")) {
		cancel(c, responses.ERROR_AUTHENTICATION_FAIL)
	}
}

func cancel(c ControllerInterface, code int) {
	c.CustomAbort(200, string(responses.NewError(code)))
}
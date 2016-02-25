package routers

import (
	"github.com/deluan/gosonic/api"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	"github.com/deluan/gosonic/controllers"
)

func init() {
	ns := beego.NewNamespace("/rest",
		beego.NSRouter("/ping.view", &api.PingController{}),
		beego.NSRouter("/getLicense.view", &api.GetLicenseController{}),
		beego.NSRouter("/getMusicFolders.view", &api.GetMusicFoldersController{}),
	)
	beego.AddNamespace(ns)

	beego.Router("/", &controllers.MainController{})

	var ValidateRequest = func(ctx *context.Context) {
		api.Validate(&beego.Controller{Ctx: ctx})
	}

	beego.InsertFilter("/rest/*", beego.BeforeRouter, ValidateRequest)
	beego.ErrorController(&controllers.MainController{})
}

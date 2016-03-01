package conf

import (
	"github.com/deluan/gosonic/api"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	"github.com/deluan/gosonic/controllers"
)

func init() {
	mapEndpoints()
	mapControllers()
	mapFilters()
}

func mapEndpoints() {
	ns := beego.NewNamespace("/rest",
		beego.NSRouter("/ping.view", &api.PingController{}, "*:Get"),
		beego.NSRouter("/getLicense.view", &api.GetLicenseController{}, "*:Get"),
		beego.NSRouter("/getMusicFolders.view", &api.GetMusicFoldersController{}, "*:Get"),
	)
	beego.AddNamespace(ns)

}

func mapControllers() {
	beego.Router("/", &controllers.MainController{})
	beego.Router("/sync", &controllers.SyncController{})

	beego.ErrorController(&controllers.MainController{})
}

func mapFilters() {
	var ValidateRequest = func(ctx *context.Context) {
		api.Validate(&beego.Controller{Ctx: ctx})
	}

	beego.InsertFilter("/rest/*", beego.BeforeRouter, ValidateRequest)
}
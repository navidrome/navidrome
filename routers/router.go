package routers

import (
	"github.com/deluan/gosonic/controllers"

	"github.com/astaxie/beego"
"github.com/astaxie/beego/context"
)

func init() {
	beego.Include(
		&controllers.PingController{},
		&controllers.GetLicenseController{},
		&controllers.GetMusicFoldersController{},
	)

	var ValidateRequest = func(ctx *context.Context) {
		controllers.Validate(&beego.Controller{Ctx: ctx})
	}

	beego.InsertFilter("/rest/*", beego.BeforeRouter, ValidateRequest)
}

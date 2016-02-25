package controllers

import (
	"github.com/astaxie/beego"
	"fmt"
)

type MainController struct{ beego.Controller }


// @router / [get]
func (this *MainController) Get() {
	this.Ctx.Redirect(302, "/static/Jamstash/")
}


func (this *MainController) Error404() {
	if beego.BConfig.RunMode == beego.DEV || beego.BConfig.Log.AccessLogs {
		r := this.Ctx.Request
		devInfo := fmt.Sprintf("   | %-10s | %-40s | %-16s | %-10s |", r.Method, r.URL.Path, " ", "notmatch")
		if beego.DefaultAccessLogFilter == nil || !beego.DefaultAccessLogFilter.Filter(this.Ctx) {
			beego.Warn(devInfo)
		}
	}
	this.CustomAbort(404, "Error 404")
}

package controllers

import (
	"github.com/astaxie/beego"
	"fmt"
)

type MainController struct{ beego.Controller }


func (c *MainController) Get() {
	c.Ctx.Redirect(302, "/static/Jamstash/")
}


func (c *MainController) Error404() {
	if beego.BConfig.RunMode == beego.DEV || beego.BConfig.Log.AccessLogs {
		r := c.Ctx.Request
		devInfo := fmt.Sprintf("   | %-10s | %-40s | %-16s | %-10s |", r.Method, r.URL.Path, " ", "notmatch")
		if beego.DefaultAccessLogFilter == nil || !beego.DefaultAccessLogFilter.Filter(c.Ctx) {
			beego.Warn(devInfo)
		}
	}
	c.CustomAbort(404, "Error 404")
}

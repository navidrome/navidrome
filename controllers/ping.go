package controllers

import "github.com/astaxie/beego"

type PingController struct {
	beego.Controller
}

// @router /rest/ping.view [get]
func (this *PingController) Get() {
	this.Ctx.WriteString("<subsonic-response xmlns=\"http://subsonic.org/restapi\" status=\"ok\" version=\"1.0.0\"></subsonic-response>")
}




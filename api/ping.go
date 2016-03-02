package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
)

type PingController struct{ beego.Controller }

func (c *PingController) Get() {
	c.Ctx.Output.Body(responses.ToXML(responses.NewEmpty()))
}

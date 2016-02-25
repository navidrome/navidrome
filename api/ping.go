package api

import (
	"encoding/xml"
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
)

type PingController struct{ beego.Controller }

func (c *PingController) Get() {
	response := responses.NewEmpty()
	xmlBody, _ := xml.Marshal(response)
	c.Ctx.Output.Body([]byte(xml.Header + string(xmlBody)))
}

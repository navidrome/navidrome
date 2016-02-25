package api

import (
	"github.com/astaxie/beego"
	"encoding/xml"
	"github.com/deluan/gosonic/api/responses"
)

type PingController struct{ beego.Controller }

func (c *PingController) Get() {
	response := responses.NewEmpty()
	xmlBody, _ := xml.Marshal(response)
	c.Ctx.Output.Body([]byte(xml.Header + string(xmlBody)))
}




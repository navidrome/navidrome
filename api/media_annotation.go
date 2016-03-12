package api

import (
	"time"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/itunesbridge"
	"github.com/deluan/gosonic/utils"
)

type MediaAnnotationController struct {
	BaseAPIController
	itunes itunesbridge.ItunesControl
}

func (c *MediaAnnotationController) Prepare() {
	utils.ResolveDependencies(&c.itunes)
}

func (c *MediaAnnotationController) Scrobble() {
	id := c.RequiredParamString("id", "Required id parameter is missing")
	time := c.ParamTime("time", time.Now())
	submission := c.ParamBool("submission", true)

	if submission {
		beego.Debug("Scrobbling", id, "at", time)
		if err := c.itunes.Scrobble(id, time); err != nil {
			beego.Error("Error scrobbling:", err)
			c.SendError(responses.ERROR_GENERIC, "Internal error")
		}
	}

	response := c.NewEmpty()
	c.SendResponse(response)
}

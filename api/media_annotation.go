package api

import (
	"fmt"
	"time"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/itunesbridge"
	"github.com/deluan/gosonic/utils"
)

type MediaAnnotationController struct {
	BaseAPIController
	itunes itunesbridge.ItunesControl
	mfRepo domain.MediaFileRepository
}

func (c *MediaAnnotationController) Prepare() {
	utils.ResolveDependencies(&c.itunes, &c.mfRepo)
}

func (c *MediaAnnotationController) Scrobble() {
	id := c.RequiredParamString("id", "Required id parameter is missing")
	time := c.ParamTime("time", time.Now())
	submission := c.ParamBool("submission", true)

	if submission {
		mf, err := c.mfRepo.Get(id)
		if err != nil || mf == nil {
			beego.Error("Id", id, "not found!")
			c.SendError(responses.ERROR_DATA_NOT_FOUND, "Id not found")
		}

		beego.Info(fmt.Sprintf(`Scrobbling (%s) "%s" at %v`, id, mf.Title, time))
		if err := c.itunes.Scrobble(id, time); err != nil {
			beego.Error("Error scrobbling:", err)
			c.SendError(responses.ERROR_GENERIC, "Internal error")
		}
	}

	response := c.NewEmpty()
	c.SendResponse(response)
}

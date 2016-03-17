package api

import (
	"fmt"
	"time"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/utils"
)

type MediaAnnotationController struct {
	BaseAPIController
	scrobbler engine.Scrobbler
}

func (c *MediaAnnotationController) Prepare() {
	utils.ResolveDependencies(&c.scrobbler)
}

func (c *MediaAnnotationController) Scrobble() {
	id := c.RequiredParamString("id", "Required id parameter is missing")
	time := c.ParamTime("time", time.Now())
	submission := c.ParamBool("submission", false)

	playerId := 1 // TODO Multiple players, based on playerName/username/clientIP(?)
	playerName := c.ParamString("c")
	username := c.ParamString("u")

	skip, err := c.scrobbler.DetectSkipped(playerId, id, submission)
	if err {
		beego.Error("Error detecting skip:", err)
	}
	if skip {
		beego.Info("Skipped previous song")
	}

	if submission {
		mf, err := c.scrobbler.Register(playerId, id, time)
		if err != nil {
			beego.Error("Error scrobbling:", err)
			c.SendError(responses.ERROR_GENERIC, "Internal error")
		}
		beego.Info(fmt.Sprintf(`Scrobbled (%s) "%s" at %v`, id, mf.Title, time))
	} else {
		mf, err := c.scrobbler.NowPlaying(playerId, id, username, playerName)
		if err != nil {
			beego.Error("Error setting", id, "as current song:", err)
			c.SendError(responses.ERROR_GENERIC, "Internal error")
		}
		beego.Info(fmt.Sprintf(`Current Song (%s) "%s" at %v`, id, mf.Title, time))
	}

	response := c.NewEmpty()
	c.SendResponse(response)
}

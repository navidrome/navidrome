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
	ids := c.RequiredParamStrings("id", "Required id parameter is missing")
	times := c.ParamTimes("time")
	if len(times) > 0 && len(times) != len(ids) {
		c.SendError(responses.ERROR_GENERIC, fmt.Sprintf("Wrong number of timestamps: %d", len(times)))
	}
	submission := c.ParamBool("submission", true)
	playerId := 1 // TODO Multiple players, based on playerName/username/clientIP(?)
	playerName := c.ParamString("c")
	username := c.ParamString("u")

	beego.Debug("Scrobbling ids:", ids, "times:", times, "submission:", submission)
	for i := range ids {
		var t time.Time
		if len(times) > 0 {
			t = times[i]
		} else {
			t = time.Now()
		}
		// TODO Fix skipped songs
		//skip, err := c.scrobbler.DetectSkipped(playerId, id, submission)
		//if err != nil {
		//	beego.Error("Error detecting skip:", err)
		//}
		//if skip {
		//	beego.Info("Skipped previous song")
		//}
		if submission {
			mf, err := c.scrobbler.Register(playerId, ids[i], t)
			if err != nil {
				beego.Error("Error scrobbling:", err)
				c.SendError(responses.ERROR_GENERIC, "Internal error")
			}
			beego.Info(fmt.Sprintf(`Scrobbled (%s) "%s" at %v`, ids[i], mf.Title, t))
		} else {
			mf, err := c.scrobbler.NowPlaying(playerId, ids[i], username, playerName)
			if err != nil {
				beego.Error("Error setting", ids[i], "as current song:", err)
				c.SendError(responses.ERROR_GENERIC, "Internal error")
			}
			beego.Info(fmt.Sprintf(`Current Song (%s) "%s" at %v`, ids[i], mf.Title, t))
		}
	}
	response := c.NewEmpty()
	c.SendResponse(response)
}

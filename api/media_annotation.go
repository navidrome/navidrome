package api

import (
	"fmt"
	"time"

	"github.com/astaxie/beego"
	"github.com/cloudsonic/sonic-server/api/responses"
	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/utils"
)

type MediaAnnotationController struct {
	BaseAPIController
	scrobbler engine.Scrobbler
	ratings   engine.Ratings
}

func (c *MediaAnnotationController) Prepare() {
	utils.ResolveDependencies(&c.scrobbler, &c.ratings)
}

func (c *MediaAnnotationController) SetRating() {
	id := c.RequiredParamString("id", "Required id parameter is missing")
	rating := c.RequiredParamInt("rating", "Required rating parameter is missing")

	beego.Debug("Setting rating", rating, "for id", id)
	err := c.ratings.SetRating(id, rating)

	switch {
	case err == domain.ErrNotFound:
		beego.Error(err)
		c.SendError(responses.ErrorDataNotFound, "Id not found")
	case err != nil:
		beego.Error(err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
	}

	c.SendEmptyResponse()
}

func (c *MediaAnnotationController) getIds() []string {
	ids := c.ParamStrings("id")
	albumIds := c.ParamStrings("albumId")

	if len(ids) == 0 && len(albumIds) == 0 {
		c.SendError(responses.ErrorMissingParameter, "Required id parameter is missing")
	}

	return append(ids, albumIds...)
}

func (c *MediaAnnotationController) Star() {
	ids := c.getIds()
	beego.Debug("Starring ids:", ids)
	err := c.ratings.SetStar(true, ids...)
	switch {
	case err == domain.ErrNotFound:
		beego.Error(err)
		c.SendError(responses.ErrorDataNotFound, "Id not found")
	case err != nil:
		beego.Error(err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
	}

	c.SendEmptyResponse()
}

func (c *MediaAnnotationController) Unstar() {
	ids := c.getIds()
	beego.Debug("Unstarring ids:", ids)
	err := c.ratings.SetStar(false, ids...)
	switch {
	case err == domain.ErrNotFound:
		beego.Error(err)
		c.SendError(responses.ErrorDataNotFound, "Directory not found")
	case err != nil:
		beego.Error(err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
	}

	c.SendEmptyResponse()
}

func (c *MediaAnnotationController) Scrobble() {
	ids := c.RequiredParamStrings("id", "Required id parameter is missing")
	times := c.ParamTimes("time")
	if len(times) > 0 && len(times) != len(ids) {
		c.SendError(responses.ErrorGeneric, "Wrong number of timestamps: %d", len(times))
	}
	submission := c.ParamBool("submission", true)
	playerId := 1 // TODO Multiple players, based on playerName/username/clientIP(?)
	playerName := c.ParamString("c")
	username := c.ParamString("u")

	beego.Debug("Scrobbling ids:", ids, "times:", times, "submission:", submission)
	for i, id := range ids {
		var t time.Time
		if len(times) > 0 {
			t = times[i]
		} else {
			t = time.Now()
		}
		if submission {
			mf, err := c.scrobbler.Register(playerId, id, t)
			if err != nil {
				beego.Error("Error scrobbling", id, "-", err)
				continue
			}
			beego.Info(fmt.Sprintf(`Scrobbled (%s) "%s" at %v`, id, mf.Title, t))
		} else {
			mf, err := c.scrobbler.NowPlaying(playerId, playerName, id, username)
			if err != nil {
				beego.Error("Error setting", id, "as current song:", err)
				continue
			}
			beego.Info(fmt.Sprintf(`Now Playing (%s) "%s" at %v`, id, mf.Title, t))
		}
	}
	c.SendEmptyResponse()
}

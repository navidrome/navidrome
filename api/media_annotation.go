package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/astaxie/beego"
	"github.com/cloudsonic/sonic-server/api/responses"
	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/engine"
)

type MediaAnnotationController struct {
	scrobbler engine.Scrobbler
	ratings   engine.Ratings
}

func NewMediaAnnotationController(scrobbler engine.Scrobbler, ratings engine.Ratings) *MediaAnnotationController {
	return &MediaAnnotationController{
		scrobbler: scrobbler,
		ratings:   ratings,
	}
}

func (c *MediaAnnotationController) SetRating(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id, err := RequiredParamString(r, "id", "Required id parameter is missing")
	if err != nil {
		return nil, err
	}
	rating, err := RequiredParamInt(r, "rating", "Required rating parameter is missing")
	if err != nil {
		return nil, err
	}

	beego.Debug("Setting rating", rating, "for id", id)
	err = c.ratings.SetRating(id, rating)

	switch {
	case err == domain.ErrNotFound:
		beego.Error(err)
		return nil, NewError(responses.ErrorDataNotFound, "Id not found")
	case err != nil:
		beego.Error(err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	return NewEmpty(), nil
}

func (c *MediaAnnotationController) getIds(r *http.Request) ([]string, error) {
	ids := ParamStrings(r, "id")
	albumIds := ParamStrings(r,"albumId")

	if len(ids) == 0 && len(albumIds) == 0 {
		return nil, NewError(responses.ErrorMissingParameter, "Required id parameter is missing")
	}

	return append(ids, albumIds...), nil
}

func (c *MediaAnnotationController) Star(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ids, err := c.getIds(r)
	if err != nil {
		return nil, err
	}
	beego.Debug("Starring ids:", ids)
	err = c.ratings.SetStar(true, ids...)
	switch {
	case err == domain.ErrNotFound:
		beego.Error(err)
		return nil, NewError(responses.ErrorDataNotFound, "Id not found")
	case err != nil:
		beego.Error(err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	return NewEmpty(), nil
}

func (c *MediaAnnotationController) Unstar(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ids, err := c.getIds(r)
	if err != nil {
		return nil, err
	}
	beego.Debug("Unstarring ids:", ids)
	err = c.ratings.SetStar(false, ids...)
	switch {
	case err == domain.ErrNotFound:
		beego.Error(err)
		return nil, NewError(responses.ErrorDataNotFound, "Directory not found")
	case err != nil:
		beego.Error(err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	return NewEmpty(), nil
}

func (c *MediaAnnotationController) Scrobble(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ids, err := RequiredParamStrings(r, "id", "Required id parameter is missing")
	if err != nil {
		return nil, err
	}
	times := ParamTimes(r, "time")
	if len(times) > 0 && len(times) != len(ids) {
		return nil, NewError(responses.ErrorGeneric, "Wrong number of timestamps: %d, should be %d", len(times), len(ids))
	}
	submission := ParamBool(r, "submission", true)
	playerId := 1 // TODO Multiple players, based on playerName/username/clientIP(?)
	playerName := ParamString(r, "c")
	username := ParamString(r, "u")

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
	return NewEmpty(), nil
}

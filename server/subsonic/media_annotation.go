package subsonic

import (
	"net/http"
	"time"

	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/model"
	"github.com/cloudsonic/sonic-server/server/subsonic/responses"
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

	log.Debug(r, "Setting rating", "rating", rating, "id", id)
	err = c.ratings.SetRating(r.Context(), id, rating)

	switch {
	case err == model.ErrNotFound:
		log.Error(r, err)
		return nil, NewError(responses.ErrorDataNotFound, "ID not found")
	case err != nil:
		log.Error(r, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	return NewResponse(), nil
}

func (c *MediaAnnotationController) getIds(r *http.Request) ([]string, error) {
	ids := ParamStrings(r, "id")
	albumIds := ParamStrings(r, "albumId")
	artistIds := ParamStrings(r, "artistId")

	if len(ids)+len(albumIds)+len(artistIds) == 0 {
		return nil, NewError(responses.ErrorMissingParameter, "Required id parameter is missing")
	}

	ids = append(ids, albumIds...)
	ids = append(ids, artistIds...)
	return ids, nil
}

func (c *MediaAnnotationController) Star(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ids, err := c.getIds(r)
	if err != nil {
		return nil, err
	}
	log.Debug(r, "Starring items", "ids", ids)
	err = c.ratings.SetStar(r.Context(), true, ids...)
	switch {
	case err == model.ErrNotFound:
		log.Error(r, err)
		return nil, NewError(responses.ErrorDataNotFound, "ID not found")
	case err != nil:
		log.Error(r, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	return NewResponse(), nil
}

func (c *MediaAnnotationController) Unstar(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ids, err := c.getIds(r)
	if err != nil {
		return nil, err
	}
	log.Debug(r, "Unstarring items", "ids", ids)
	err = c.ratings.SetStar(r.Context(), false, ids...)
	switch {
	case err == model.ErrNotFound:
		log.Error(r, err)
		return nil, NewError(responses.ErrorDataNotFound, "Directory not found")
	case err != nil:
		log.Error(r, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	return NewResponse(), nil
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

	log.Debug(r, "Scrobbling tracks", "ids", ids, "times", times, "submission", submission)
	for i, id := range ids {
		var t time.Time
		if len(times) > 0 {
			t = times[i]
		} else {
			t = time.Now()
		}
		if submission {
			mf, err := c.scrobbler.Register(r.Context(), playerId, id, t)
			if err != nil {
				log.Error(r, "Error scrobbling track", "id", id, err)
				continue
			}
			log.Info(r, "Scrobbled", "id", id, "title", mf.Title, "timestamp", t)
		} else {
			mf, err := c.scrobbler.NowPlaying(r.Context(), playerId, playerName, id, username)
			if err != nil {
				log.Error(r, "Error setting current song", "id", id, err)
				continue
			}
			log.Info(r, "Now Playing", "id", id, "title", mf.Title, "timestamp", t)
		}
	}
	return NewResponse(), nil
}

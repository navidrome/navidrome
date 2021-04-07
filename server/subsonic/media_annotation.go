package subsonic

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils"
)

type MediaAnnotationController struct {
	ds     model.DataStore
	npRepo core.NowPlaying
}

func NewMediaAnnotationController(ds model.DataStore, npr core.NowPlaying) *MediaAnnotationController {
	return &MediaAnnotationController{ds: ds, npRepo: npr}
}

func (c *MediaAnnotationController) SetRating(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id, err := requiredParamString(r, "id")
	if err != nil {
		return nil, err
	}
	rating, err := requiredParamInt(r, "rating")
	if err != nil {
		return nil, err
	}

	log.Debug(r, "Setting rating", "rating", rating, "id", id)
	err = c.setRating(r.Context(), id, rating)

	switch {
	case err == model.ErrNotFound:
		log.Error(r, err)
		return nil, newError(responses.ErrorDataNotFound, "ID not found")
	case err != nil:
		log.Error(r, err)
		return nil, err
	}

	return newResponse(), nil
}

func (c *MediaAnnotationController) setRating(ctx context.Context, id string, rating int) error {
	var exist bool
	var err error

	if exist, err = c.ds.Artist(ctx).Exists(id); err != nil {
		return err
	} else if exist {
		return c.ds.Artist(ctx).SetRating(rating, id)
	}

	if exist, err = c.ds.Album(ctx).Exists(id); err != nil {
		return err
	} else if exist {
		return c.ds.Album(ctx).SetRating(rating, id)
	}

	return c.ds.MediaFile(ctx).SetRating(rating, id)
}

func (c *MediaAnnotationController) Star(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ids := utils.ParamStrings(r, "id")
	albumIds := utils.ParamStrings(r, "albumId")
	artistIds := utils.ParamStrings(r, "artistId")
	if len(ids)+len(albumIds)+len(artistIds) == 0 {
		return nil, newError(responses.ErrorMissingParameter, "Required id parameter is missing")
	}
	ids = append(ids, albumIds...)
	ids = append(ids, artistIds...)

	err := c.setStar(r.Context(), true, ids...)
	if err != nil {
		return nil, err
	}

	return newResponse(), nil
}

func (c *MediaAnnotationController) Unstar(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ids := utils.ParamStrings(r, "id")
	albumIds := utils.ParamStrings(r, "albumId")
	artistIds := utils.ParamStrings(r, "artistId")
	if len(ids)+len(albumIds)+len(artistIds) == 0 {
		return nil, newError(responses.ErrorMissingParameter, "Required id parameter is missing")
	}
	ids = append(ids, albumIds...)
	ids = append(ids, artistIds...)

	err := c.setStar(r.Context(), false, ids...)
	if err != nil {
		return nil, err
	}

	return newResponse(), nil
}

func (c *MediaAnnotationController) Scrobble(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ids, err := requiredParamStrings(r, "id")
	if err != nil {
		return nil, err
	}
	times := utils.ParamTimes(r, "time")
	if len(times) > 0 && len(times) != len(ids) {
		return nil, newError(responses.ErrorGeneric, "Wrong number of timestamps: %d, should be %d", len(times), len(ids))
	}
	submission := utils.ParamBool(r, "submission", true)
	playerId := 1 // TODO Multiple players, based on playerName/username/clientIP(?)
	playerName := utils.ParamString(r, "c")
	username := utils.ParamString(r, "u")

	log.Debug(r, "Scrobbling tracks", "ids", ids, "times", times, "submission", submission)
	for i, id := range ids {
		var t time.Time
		if len(times) > 0 {
			t = times[i]
		} else {
			t = time.Now()
		}
		if submission {
			_, err := c.scrobblerRegister(r.Context(), playerId, id, t)
			if err != nil {
				log.Error(r, "Error scrobbling track", "id", id, err)
				continue
			}
		} else {
			_, err := c.scrobblerNowPlaying(r.Context(), playerId, playerName, id, username)
			if err != nil {
				log.Error(r, "Error setting current song", "id", id, err)
				continue
			}
		}
	}
	return newResponse(), nil
}

func (c *MediaAnnotationController) scrobblerRegister(ctx context.Context, playerId int, trackId string, playTime time.Time) (*model.MediaFile, error) {
	var mf *model.MediaFile
	var err error
	err = c.ds.WithTx(func(tx model.DataStore) error {
		mf, err = c.ds.MediaFile(ctx).Get(trackId)
		if err != nil {
			return err
		}
		err = c.ds.MediaFile(ctx).IncPlayCount(trackId, playTime)
		if err != nil {
			return err
		}
		err = c.ds.Album(ctx).IncPlayCount(mf.AlbumID, playTime)
		if err != nil {
			return err
		}
		err = c.ds.Artist(ctx).IncPlayCount(mf.ArtistID, playTime)
		return err
	})

	username, _ := request.UsernameFrom(ctx)
	if err != nil {
		log.Error("Error while scrobbling", "trackId", trackId, "user", username, err)
	} else {
		log.Info("Scrobbled", "title", mf.Title, "artist", mf.Artist, "user", username)
	}

	return mf, err
}

func (c *MediaAnnotationController) scrobblerNowPlaying(ctx context.Context, playerId int, playerName, trackId, username string) (*model.MediaFile, error) {
	mf, err := c.ds.MediaFile(ctx).Get(trackId)
	if err != nil {
		return nil, err
	}

	if mf == nil {
		return nil, fmt.Errorf(`ID "%s" not found`, trackId)
	}

	log.Info("Now Playing", "title", mf.Title, "artist", mf.Artist, "user", username)

	info := &core.NowPlayingInfo{TrackID: trackId, Username: username, Start: time.Now(), PlayerId: playerId, PlayerName: playerName}
	return mf, c.npRepo.Enqueue(info)
}

func (c *MediaAnnotationController) setStar(ctx context.Context, star bool, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	log.Debug(ctx, "Changing starred", "ids", ids, "starred", star)
	if len(ids) == 0 {
		log.Warn(ctx, "Cannot star/unstar an empty list of ids")
		return nil
	}

	err := c.ds.WithTx(func(tx model.DataStore) error {
		for _, id := range ids {
			exist, err := tx.Album(ctx).Exists(id)
			if err != nil {
				return err
			}
			if exist {
				err = tx.Album(ctx).SetStar(star, ids...)
				if err != nil {
					return err
				}
				continue
			}
			exist, err = tx.Artist(ctx).Exists(id)
			if err != nil {
				return err
			}
			if exist {
				err = tx.Artist(ctx).SetStar(star, ids...)
				if err != nil {
					return err
				}
				continue
			}
			err = tx.MediaFile(ctx).SetStar(star, ids...)
			if err != nil {
				return err
			}
		}
		return nil
	})

	switch {
	case err == model.ErrNotFound:
		log.Error(ctx, err)
		return newError(responses.ErrorDataNotFound, "ID not found")
	case err != nil:
		log.Error(ctx, err)
		return err
	}
	return nil
}

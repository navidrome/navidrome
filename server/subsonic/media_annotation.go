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
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils"
)

type MediaAnnotationController struct {
	ds     model.DataStore
	npRepo core.NowPlaying
	broker events.Broker
}

func NewMediaAnnotationController(ds model.DataStore, npr core.NowPlaying, broker events.Broker) *MediaAnnotationController {
	return &MediaAnnotationController{ds: ds, npRepo: npr, broker: broker}
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
	var repo model.AnnotatedRepository
	var resource string

	entity, err := core.GetEntityByID(ctx, c.ds, id)
	if err != nil {
		return err
	}
	switch entity.(type) {
	case *model.Artist:
		repo = c.ds.Artist(ctx)
		resource = "artist"
	case *model.Album:
		repo = c.ds.Album(ctx)
		resource = "album"
	default:
		repo = c.ds.MediaFile(ctx)
		resource = "song"
	}
	err = repo.SetRating(rating, id)
	if err != nil {
		return err
	}
	event := &events.RefreshResource{}
	c.broker.SendMessage(ctx, event.With(resource, id))
	return nil
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
	ctx := r.Context()
	event := &events.RefreshResource{}
	submissions := 0

	log.Debug(r, "Scrobbling tracks", "ids", ids, "times", times, "submission", submission)
	for i, id := range ids {
		var t time.Time
		if len(times) > 0 {
			t = times[i]
		} else {
			t = time.Now()
		}
		if submission {
			mf, err := c.scrobblerRegister(ctx, playerId, id, t)
			if err != nil {
				log.Error(r, "Error scrobbling track", "id", id, err)
				continue
			}
			submissions++
			event.With("song", mf.ID).With("album", mf.AlbumID).With("artist", mf.AlbumArtistID)
		} else {
			_, err := c.scrobblerNowPlaying(ctx, playerId, playerName, id, username)
			if err != nil {
				log.Error(r, "Error setting current song", "id", id, err)
				continue
			}
		}
	}
	if submissions > 0 {
		c.broker.SendMessage(ctx, event)
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
	event := &events.RefreshResource{}
	err := c.ds.WithTx(func(tx model.DataStore) error {
		for _, id := range ids {
			exist, err := tx.Album(ctx).Exists(id)
			if err != nil {
				return err
			}
			if exist {
				err = tx.Album(ctx).SetStar(star, id)
				if err != nil {
					return err
				}
				event = event.With("album", id)
				continue
			}
			exist, err = tx.Artist(ctx).Exists(id)
			if err != nil {
				return err
			}
			if exist {
				err = tx.Artist(ctx).SetStar(star, id)
				if err != nil {
					return err
				}
				event = event.With("artist", id)
				continue
			}
			err = tx.MediaFile(ctx).SetStar(star, id)
			if err != nil {
				return err
			}
			event = event.With("song", id)
		}
		c.broker.SendMessage(ctx, event)
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

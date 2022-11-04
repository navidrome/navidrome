package subsonic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils"
)

type MediaAnnotationController struct {
	ds          model.DataStore
	playTracker scrobbler.PlayTracker
	broker      events.Broker
}

func NewMediaAnnotationController(ds model.DataStore, playTracker scrobbler.PlayTracker, broker events.Broker) *MediaAnnotationController {
	return &MediaAnnotationController{ds: ds, playTracker: playTracker, broker: broker}
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

	if errors.Is(err, model.ErrNotFound) {
		log.Error(r, err)
		return nil, newError(responses.ErrorDataNotFound, "ID not found")
	}
	if err != nil {
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

	if errors.Is(err, model.ErrNotFound) {
		log.Error(ctx, err)
		return newError(responses.ErrorDataNotFound, "ID not found")
	}
	if err != nil {
		log.Error(ctx, err)
		return err
	}
	return nil
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
	ctx := r.Context()

	if submission {
		err := c.scrobblerSubmit(ctx, ids, times)
		if err != nil {
			log.Error(ctx, "Error registering scrobbles", "ids", ids, "times", times, err)
		}
	} else {
		err := c.scrobblerNowPlaying(ctx, ids[0])
		if err != nil {
			log.Error(ctx, "Error setting NowPlaying", "id", ids[0], err)
		}
	}

	return newResponse(), nil
}

func (c *MediaAnnotationController) scrobblerSubmit(ctx context.Context, ids []string, times []time.Time) error {
	var submissions []scrobbler.Submission
	log.Debug(ctx, "Scrobbling tracks", "ids", ids, "times", times)
	for i, id := range ids {
		var t time.Time
		if len(times) > 0 {
			t = times[i]
		} else {
			t = time.Now()
		}
		submissions = append(submissions, scrobbler.Submission{TrackID: id, Timestamp: t})
	}

	return c.playTracker.Submit(ctx, submissions)
}

func (c *MediaAnnotationController) scrobblerNowPlaying(ctx context.Context, trackId string) error {
	mf, err := c.ds.MediaFile(ctx).Get(trackId)
	if err != nil {
		return err
	}
	if mf == nil {
		return fmt.Errorf(`ID "%s" not found`, trackId)
	}

	player, _ := request.PlayerFrom(ctx)
	username, _ := request.UsernameFrom(ctx)
	client, _ := request.ClientFrom(ctx)
	clientId, ok := request.ClientUniqueIdFrom(ctx)
	if !ok {
		clientId = player.ID
	}

	log.Info(ctx, "Now Playing", "title", mf.Title, "artist", mf.Artist, "user", username, "player", player.Name)
	err = c.playTracker.NowPlaying(ctx, clientId, client, trackId)
	return err
}

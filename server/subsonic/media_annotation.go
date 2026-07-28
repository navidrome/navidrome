package subsonic

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
	"github.com/navidrome/navidrome/utils/slice"
)

func (api *Router) SetRating(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	rating, err := p.Int("rating")
	if err != nil {
		return nil, err
	}

	log.Debug(r, "Setting rating", "rating", rating, "id", id)
	err = api.setRating(r.Context(), id, rating)
	if err != nil {
		log.Error(r, err)
		return nil, err
	}

	return newResponse(), nil
}

func (api *Router) setRating(ctx context.Context, id string, rating int) error {
	var repo model.AnnotatedRepository
	var resource string

	entity, err := model.GetEntityByID(ctx, api.ds, id)
	if err != nil {
		return err
	}
	switch entity.(type) {
	case *model.Artist:
		repo = api.ds.Artist(ctx)
		resource = "artist"
	case *model.Album:
		repo = api.ds.Album(ctx)
		resource = "album"
	default:
		repo = api.ds.MediaFile(ctx)
		resource = "song"
	}
	err = repo.SetRating(rating, id)
	if err != nil {
		return err
	}
	event := &events.RefreshResource{}
	api.broker.SendMessage(ctx, event.With(resource, id))
	return nil
}

func (api *Router) Star(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	ids, _ := p.Strings("id")
	albumIds, _ := p.Strings("albumId")
	artistIds, _ := p.Strings("artistId")
	if len(ids)+len(albumIds)+len(artistIds) == 0 {
		return nil, newError(responses.ErrorMissingParameter, "Required id parameter is missing")
	}
	ids = append(ids, albumIds...)
	ids = append(ids, artistIds...)

	err := api.setStar(r.Context(), true, ids...)
	if err != nil {
		return nil, err
	}

	return newResponse(), nil
}

func (api *Router) Unstar(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	ids, _ := p.Strings("id")
	albumIds, _ := p.Strings("albumId")
	artistIds, _ := p.Strings("artistId")
	if len(ids)+len(albumIds)+len(artistIds) == 0 {
		return nil, newError(responses.ErrorMissingParameter, "Required id parameter is missing")
	}
	ids = append(ids, albumIds...)
	ids = append(ids, artistIds...)

	err := api.setStar(r.Context(), false, ids...)
	if err != nil {
		return nil, err
	}

	return newResponse(), nil
}

func (api *Router) setStar(ctx context.Context, star bool, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	log.Debug(ctx, "Changing starred", "ids", ids, "starred", star)
	if len(ids) == 0 {
		log.Warn(ctx, "Cannot star/unstar an empty list of ids")
		return nil
	}
	event := &events.RefreshResource{}
	err := api.ds.WithTxImmediate(func(tx model.DataStore) error {
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
		api.broker.SendMessage(ctx, event)
		return nil
	})
	if err != nil {
		log.Error(ctx, err)
		return err
	}
	return nil
}

func (api *Router) Skip(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	ids, err := p.Strings("id")
	if err != nil || len(ids) == 0 {
		return nil, newError(responses.ErrorMissingParameter, "Required id parameter is missing")
	}

	err = api.setSkip(r.Context(), true, ids...)
	if err != nil {
		return nil, err
	}

	return newResponse(), nil
}

func (api *Router) Unskip(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	ids, err := p.Strings("id")
	if err != nil || len(ids) == 0 {
		return nil, newError(responses.ErrorMissingParameter, "Required id parameter is missing")
	}

	err = api.setSkip(r.Context(), false, ids...)
	if err != nil {
		return nil, err
	}

	return newResponse(), nil
}

func (api *Router) setSkip(ctx context.Context, skip bool, ids ...string) error {
	log.Debug(ctx, "Changing skipped", "ids", ids, "skipped", skip)
	event := &events.RefreshResource{}
	err := api.ds.WithTxImmediate(func(tx model.DataStore) error {
		for _, id := range ids {
			err := tx.MediaFile(ctx).SetSkip(skip, id)
			if err != nil {
				return err
			}
			event = event.With("song", id)
		}
		api.broker.SendMessage(ctx, event)
		return nil
	})
	if err != nil {
		log.Error(ctx, err)
		return err
	}
	return nil
}

func (api *Router) SetUserTag(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	tag, err := p.String("tag")
	if err != nil {
		return nil, err
	}

	log.Debug(r, "Setting user tag", "id", id, "tag", tag)
	// Subsonic's *UserTag* family is only ever called by the AI Auto-Tagging
	// plugin (there's no human-facing UI on this API surface) - hardcoding
	// the source here means that plugin needs no changes at all.
	err = api.ds.MediaFileTag(r.Context()).TagSong(id, tag, model.MediaFileTagSourceAI)
	if err != nil {
		log.Error(r, err)
		return nil, err
	}

	return newResponse(), nil
}

func (api *Router) RemoveUserTag(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	tag, err := p.String("tag")
	if err != nil {
		return nil, err
	}

	log.Debug(r, "Removing user tag", "id", id, "tag", tag)
	err = api.ds.MediaFileTag(r.Context()).UntagSong(id, tag)
	if err != nil {
		log.Error(r, err)
		return nil, err
	}

	return newResponse(), nil
}

func (api *Router) GetUserTags(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}

	// AI-only: this is what AI Auto-Tagging's "already tagged, skip it" scan
	// check relies on. If it also saw a human's manually-added My Tag here,
	// a track a person tagged before AI ever ran would be wrongly skipped.
	tags, err := api.ds.MediaFileTag(r.Context()).TagsForSong(id, model.MediaFileTagSourceAI)
	if err != nil {
		log.Error(r, err)
		return nil, err
	}

	res := newResponse()
	res.UserTags = &responses.UserTags{Tag: tags}
	return res, nil
}

// GetAllUserTags returns every distinct AI-written tag name the caller has
// applied to any song, for callers (e.g. plugins) that need to discover what
// tags exist without scanning the whole library.
func (api *Router) GetAllUserTags(r *http.Request) (*responses.Subsonic, error) {
	tags, err := api.ds.MediaFileTag(r.Context()).AllTagNames(model.MediaFileTagSourceAI)
	if err != nil {
		log.Error(r, err)
		return nil, err
	}

	res := newResponse()
	res.UserTags = &responses.UserTags{Tag: tags}
	return res, nil
}

// GetSongsByUserTag returns every song the caller has tagged (via AI Auto-
// Tagging) with the given tag name, with full metadata - avoids requiring a
// separate getSong.view call per matching track.
func (api *Router) GetSongsByUserTag(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	tag, err := p.String("tag")
	if err != nil {
		return nil, err
	}

	ctx := r.Context()
	ids, err := api.ds.MediaFileTag(ctx).SongIDsForTag(tag, model.MediaFileTagSourceAI)
	if err != nil {
		log.Error(r, err)
		return nil, err
	}

	res := newResponse()
	if len(ids) == 0 {
		res.SongsByUserTag = &responses.Songs{}
		return res, nil
	}

	mfs, err := api.ds.MediaFile(ctx).GetAll(model.QueryOptions{Filters: Eq{"media_file.id": ids}})
	if err != nil {
		log.Error(r, err)
		return nil, err
	}

	res.SongsByUserTag = &responses.Songs{Songs: slice.MapWithArg(mfs, ctx, childFromMediaFile)}
	return res, nil
}

// GetAllMyTags returns every distinct tag name the caller has hand-applied
// themselves (source=user - the native REST /mediaFileTag "My Tags" feature),
// so a client like Cirque can read them over Subsonic the same way it already
// reads the AI-only UserTag family, without either family ever mixing with
// the other.
func (api *Router) GetAllMyTags(r *http.Request) (*responses.Subsonic, error) {
	tags, err := api.ds.MediaFileTag(r.Context()).AllTagNames(model.MediaFileTagSourceUser)
	if err != nil {
		log.Error(r, err)
		return nil, err
	}

	res := newResponse()
	res.MyTags = &responses.UserTags{Tag: tags}
	return res, nil
}

// GetSongsByMyTag returns every song the caller has hand-tagged (source=user)
// with the given tag name, with full metadata - the My Tags counterpart to
// GetSongsByUserTag.
func (api *Router) GetSongsByMyTag(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	tag, err := p.String("tag")
	if err != nil {
		return nil, err
	}

	ctx := r.Context()
	ids, err := api.ds.MediaFileTag(ctx).SongIDsForTag(tag, model.MediaFileTagSourceUser)
	if err != nil {
		log.Error(r, err)
		return nil, err
	}

	res := newResponse()
	if len(ids) == 0 {
		res.SongsByMyTag = &responses.Songs{}
		return res, nil
	}

	mfs, err := api.ds.MediaFile(ctx).GetAll(model.QueryOptions{Filters: Eq{"media_file.id": ids}})
	if err != nil {
		log.Error(r, err)
		return nil, err
	}

	res.SongsByMyTag = &responses.Songs{Songs: slice.MapWithArg(mfs, ctx, childFromMediaFile)}
	return res, nil
}

func (api *Router) Scrobble(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	ids, err := p.Strings("id")
	if err != nil {
		return nil, err
	}
	times, _ := p.Times("time")
	if len(times) > 0 && len(times) != len(ids) {
		return nil, newError(responses.ErrorGeneric, "Wrong number of timestamps: %d, should be %d", len(times), len(ids))
	}
	submission := p.BoolOr("submission", true)
	position := p.IntOr("position", 0)

	source := p.StringOr("source", "")
	origin := p.StringOr("origin", "")
	playbackMode := p.StringOr("playbackMode", "")

	ctx := r.Context()

	if submission {
		err := api.scrobblerSubmit(ctx, ids, times, source, origin, playbackMode)
		if err != nil {
			log.Error(ctx, "Error registering scrobbles", "ids", ids, "times", times, err)
		}
	} else {
		err := api.scrobblerNowPlaying(ctx, ids[0], position, source, origin, playbackMode)
		if err != nil {
			log.Error(ctx, "Error setting NowPlaying", "id", ids[0], err)
		}
	}

	return newResponse(), nil
}

func (api *Router) scrobblerSubmit(ctx context.Context, ids []string, times []time.Time, source, origin, playbackMode string) error {
	var submissions []scrobbler.Submission
	log.Debug(ctx, "Scrobbling tracks", "ids", ids, "times", times)
	for i, id := range ids {
		var t time.Time
		if len(times) > 0 {
			t = times[i]
		} else {
			t = time.Now()
		}
		submissions = append(submissions, scrobbler.Submission{
			TrackID:      id,
			Timestamp:    t,
			Source:       source,
			Origin:       origin,
			PlaybackMode: playbackMode,
		})
	}

	return api.scrobbler.Submit(ctx, submissions)
}

func (api *Router) scrobblerNowPlaying(ctx context.Context, trackId string, position int, source, origin, playbackMode string) error {
	mf, err := api.ds.MediaFile(ctx).Get(trackId)
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

	log.Info(ctx, "Now Playing", "title", mf.Title, "artist", mf.Artist, "user", username, "player", player.Name, "position", position)
	return api.scrobbler.ReportPlayback(ctx, scrobbler.ReportPlaybackParams{
		MediaId:      trackId,
		PositionMs:   int64(position) * 1000,
		State:        scrobbler.StatePlaying,
		PlaybackRate: 1.0,
		ClientId:     clientId,
		ClientName:   client,
		Source:       source,
		Origin:       origin,
		PlaybackMode: playbackMode,
	})
}

func (api *Router) ReportPlayback(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	mediaId, err := p.String("mediaId")
	if err != nil {
		return nil, err
	}
	mediaType, err := p.String("mediaType")
	if err != nil {
		return nil, err
	}
	positionMs, err := p.Int64("positionMs")
	if err != nil {
		return nil, err
	}
	if positionMs < 0 {
		return nil, newError(responses.ErrorGeneric, "positionMs must be non-negative")
	}
	state, err := p.String("state")
	if err != nil {
		return nil, err
	}

	if !scrobbler.ValidStates[state] {
		return nil, newError(responses.ErrorGeneric, "Invalid state: %s", state)
	}

	playbackRate := p.Float64Or("playbackRate", 1.0)
	if math.IsNaN(playbackRate) || math.IsInf(playbackRate, 0) || playbackRate <= 0 {
		return nil, newError(responses.ErrorGeneric, "playbackRate must be a finite positive number")
	}
	ignoreScrobble := p.BoolOr("ignoreScrobble", false)

	source := p.StringOr("source", "")
	origin := p.StringOr("origin", "")
	playbackMode := p.StringOr("playbackMode", "")

	ctx := r.Context()
	if mediaType != "song" {
		log.Warn(ctx, "reportPlayback received unsupported mediaType", "mediaType", mediaType, "mediaId", mediaId)
	}

	player, _ := request.PlayerFrom(ctx)
	client, _ := request.ClientFrom(ctx)
	clientId, ok := request.ClientUniqueIdFrom(ctx)
	if !ok {
		clientId = player.ID
	}

	err = api.scrobbler.ReportPlayback(ctx, scrobbler.ReportPlaybackParams{
		MediaId:        mediaId,
		PositionMs:     positionMs,
		State:          state,
		PlaybackRate:   playbackRate,
		IgnoreScrobble: ignoreScrobble,
		ClientId:       clientId,
		ClientName:     client,
		Source:         source,
		Origin:         origin,
		PlaybackMode:   playbackMode,
	})
	if err != nil {
		log.Error(ctx, "Error in ReportPlayback", "mediaId", mediaId, "state", state, err)
		return nil, err
	}

	return newResponse(), nil
}

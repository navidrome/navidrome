package lastfm

import (
	"context"
	"errors"
	"net/http"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
)

const (
	lastFMAgentName    = "lastfm"
	sessionKeyProperty = "LastFMSessionKey"
)

type lastfmAgent struct {
	ds          model.DataStore
	sessionKeys *agents.SessionKeys
	apiKey      string
	secret      string
	proxyStars  bool
	lang        string
	client      *Client
}

func lastFMConstructor(ds model.DataStore) *lastfmAgent {
	l := &lastfmAgent{
		ds:          ds,
		lang:        conf.Server.LastFM.Language,
		apiKey:      conf.Server.LastFM.ApiKey,
		secret:      conf.Server.LastFM.Secret,
		proxyStars:  conf.Server.LastFM.ProxyStars,
		sessionKeys: &agents.SessionKeys{DataStore: ds, KeyName: sessionKeyProperty},
	}
	hc := &http.Client{
		Timeout: consts.DefaultHttpClientTimeOut,
	}
	chc := utils.NewCachedHTTPClient(hc, consts.DefaultHttpClientTimeOut)
	l.client = NewClient(l.apiKey, l.secret, l.lang, chc)
	return l
}

func (l *lastfmAgent) AgentName() string {
	return lastFMAgentName
}

func (l *lastfmAgent) GetMBID(ctx context.Context, id string, name string) (string, error) {
	a, err := l.callArtistGetInfo(ctx, name, "")
	if err != nil {
		return "", err
	}
	if a.MBID == "" {
		return "", agents.ErrNotFound
	}
	return a.MBID, nil
}

func (l *lastfmAgent) GetURL(ctx context.Context, id, name, mbid string) (string, error) {
	a, err := l.callArtistGetInfo(ctx, name, mbid)
	if err != nil {
		return "", err
	}
	if a.URL == "" {
		return "", agents.ErrNotFound
	}
	return a.URL, nil
}

func (l *lastfmAgent) GetBiography(ctx context.Context, id, name, mbid string) (string, error) {
	a, err := l.callArtistGetInfo(ctx, name, mbid)
	if err != nil {
		return "", err
	}
	if a.Bio.Summary == "" {
		return "", agents.ErrNotFound
	}
	return a.Bio.Summary, nil
}

func (l *lastfmAgent) GetSimilar(ctx context.Context, id, name, mbid string, limit int) ([]agents.Artist, error) {
	resp, err := l.callArtistGetSimilar(ctx, name, mbid, limit)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, agents.ErrNotFound
	}
	var res []agents.Artist
	for _, a := range resp {
		res = append(res, agents.Artist{
			Name: a.Name,
			MBID: a.MBID,
		})
	}
	return res, nil
}

func (l *lastfmAgent) GetTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error) {
	resp, err := l.callArtistGetTopTracks(ctx, artistName, mbid, count)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, agents.ErrNotFound
	}
	var res []agents.Song
	for _, t := range resp {
		res = append(res, agents.Song{
			Name: t.Name,
			MBID: t.MBID,
		})
	}
	return res, nil
}

func (l *lastfmAgent) callArtistGetInfo(ctx context.Context, name string, mbid string) (*Artist, error) {
	a, err := l.client.ArtistGetInfo(ctx, name, mbid)
	var lfErr *lastFMError
	isLastFMError := errors.As(err, &lfErr)

	if mbid != "" && ((err == nil && a.Name == "[unknown]") || (isLastFMError && lfErr.Code == 6)) {
		log.Warn(ctx, "LastFM/artist.getInfo could not find artist by mbid, trying again", "artist", name, "mbid", mbid)
		return l.callArtistGetInfo(ctx, name, "")
	}

	if err != nil {
		log.Error(ctx, "Error calling LastFM/artist.getInfo", "artist", name, "mbid", mbid, err)
		return nil, err
	}
	return a, nil
}

func (l *lastfmAgent) callArtistGetSimilar(ctx context.Context, name string, mbid string, limit int) ([]Artist, error) {
	s, err := l.client.ArtistGetSimilar(ctx, name, mbid, limit)
	var lfErr *lastFMError
	isLastFMError := errors.As(err, &lfErr)
	if mbid != "" && ((err == nil && s.Attr.Artist == "[unknown]") || (isLastFMError && lfErr.Code == 6)) {
		log.Warn(ctx, "LastFM/artist.getSimilar could not find artist by mbid, trying again", "artist", name, "mbid", mbid)
		return l.callArtistGetSimilar(ctx, name, "", limit)
	}
	if err != nil {
		log.Error(ctx, "Error calling LastFM/artist.getSimilar", "artist", name, "mbid", mbid, err)
		return nil, err
	}
	return s.Artists, nil
}

func (l *lastfmAgent) callArtistGetTopTracks(ctx context.Context, artistName, mbid string, count int) ([]Track, error) {
	t, err := l.client.ArtistGetTopTracks(ctx, artistName, mbid, count)
	var lfErr *lastFMError
	isLastFMError := errors.As(err, &lfErr)
	if mbid != "" && ((err == nil && t.Attr.Artist == "[unknown]") || (isLastFMError && lfErr.Code == 6)) {
		log.Warn(ctx, "LastFM/artist.getTopTracks could not find artist by mbid, trying again", "artist", artistName, "mbid", mbid)
		return l.callArtistGetTopTracks(ctx, artistName, "", count)
	}
	if err != nil {
		log.Error(ctx, "Error calling LastFM/artist.getTopTracks", "artist", artistName, "mbid", mbid, err)
		return nil, err
	}
	return t.Track, nil
}

func (l *lastfmAgent) NowPlaying(ctx context.Context, userId string, track *model.MediaFile) error {
	sk, err := l.sessionKeys.Get(ctx, userId)
	if err != nil || sk == "" {
		return scrobbler.ErrNotAuthorized
	}

	err = l.client.UpdateNowPlaying(ctx, sk, ScrobbleInfo{
		artist:      track.Artist,
		track:       track.Title,
		album:       track.Album,
		trackNumber: track.TrackNumber,
		mbid:        track.MbzTrackID,
		duration:    int(track.Duration),
		albumArtist: track.AlbumArtist,
	})
	if err != nil {
		log.Warn(ctx, "Last.fm client.updateNowPlaying returned error", "track", track.Title, err)
		return scrobbler.ErrUnrecoverable
	}
	return nil
}

func (l *lastfmAgent) Scrobble(ctx context.Context, userId string, s scrobbler.Scrobble) error {
	sk, err := l.sessionKeys.Get(ctx, userId)
	if err != nil || sk == "" {
		return scrobbler.ErrNotAuthorized
	}

	if s.Duration <= 30 {
		log.Debug(ctx, "Skipping Last.fm scrobble for short song", "track", s.Title, "duration", s.Duration)
		return nil
	}
	err = l.client.Scrobble(ctx, sk, ScrobbleInfo{
		artist:      s.Artist,
		track:       s.Title,
		album:       s.Album,
		trackNumber: s.TrackNumber,
		mbid:        s.MbzTrackID,
		duration:    int(s.Duration),
		albumArtist: s.AlbumArtist,
		timestamp:   s.TimeStamp,
	})
	if err == nil {
		return nil
	}
	var lfErr *lastFMError
	isLastFMError := errors.As(err, &lfErr)
	if !isLastFMError {
		log.Warn(ctx, "Last.fm client.scrobble returned error", "track", s.Title, err)
		return scrobbler.ErrRetryLater
	}
	if lfErr.Code == 11 || lfErr.Code == 16 {
		return scrobbler.ErrRetryLater
	}
	return scrobbler.ErrUnrecoverable
}

func (l *lastfmAgent) IsAuthorized(ctx context.Context, userId string) bool {
	sk, err := l.sessionKeys.Get(ctx, userId)
	return err == nil && sk != ""
}

func (l *lastfmAgent) CanProxyStars(ctx context.Context, userId string) bool {
	if !l.proxyStars {
		return false
	}
	return l.IsAuthorized(ctx, userId)
}

func (l *lastfmAgent) CanStar(_ *model.MediaFile) bool {
	return true
}

func (l *lastfmAgent) Star(ctx context.Context, userId string, star bool, track *model.MediaFile) error {
	sk, err := l.sessionKeys.Get(ctx, userId)
	if err != nil || sk == "" {
		return scrobbler.ErrNotAuthorized
	}

	err = l.client.Star(ctx, sk, star, track.Artist, track.Title)

	if err == nil {
		return nil
	}
	var lfErr *lastFMError
	isLastFMError := errors.As(err, &lfErr)
	if !isLastFMError {
		log.Warn(ctx, "Last.fm client.scrobble returned error", "track", track.Title, err)
		return scrobbler.ErrRetryLater
	}
	if lfErr.Code == 11 || lfErr.Code == 16 {
		return scrobbler.ErrRetryLater
	}
	return scrobbler.ErrUnrecoverable
}

func init() {
	conf.AddHook(func() {
		if conf.Server.LastFM.Enabled {
			agents.Register(lastFMAgentName, func(ds model.DataStore) agents.Interface {
				return lastFMConstructor(ds)
			})
			scrobbler.Register(lastFMAgentName, func(ds model.DataStore) scrobbler.Scrobbler {
				return lastFMConstructor(ds)
			})
		}
	})
}

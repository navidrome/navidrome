package listenbrainz

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
	listenBrainzAgentName = "listenbrainz"
	sessionKeyProperty    = "ListenBrainzSessionKey"
)

type listenBrainzAgent struct {
	ds          model.DataStore
	sessionKeys *agents.SessionKeys
	baseURL     string
	proxyStars  bool
	client      *Client
}

func listenBrainzConstructor(ds model.DataStore) *listenBrainzAgent {
	l := &listenBrainzAgent{
		ds:          ds,
		sessionKeys: &agents.SessionKeys{DataStore: ds, KeyName: sessionKeyProperty},
		baseURL:     conf.Server.ListenBrainz.BaseURL,
		proxyStars:  conf.Server.ListenBrainz.ProxyStars,
	}
	hc := &http.Client{
		Timeout: consts.DefaultHttpClientTimeOut,
	}
	chc := utils.NewCachedHTTPClient(hc, consts.DefaultHttpClientTimeOut)
	l.client = NewClient(l.baseURL, chc)
	return l
}

func (l *listenBrainzAgent) AgentName() string {
	return listenBrainzAgentName
}

func (l *listenBrainzAgent) formatListen(track *model.MediaFile) listenInfo {
	li := listenInfo{
		TrackMetadata: trackMetadata{
			ArtistName:  track.Artist,
			TrackName:   track.Title,
			ReleaseName: track.Album,
			AdditionalInfo: additionalInfo{
				SubmissionClient:        consts.AppName,
				SubmissionClientVersion: consts.Version,
				TrackNumber:             track.TrackNumber,
				ArtistMbzIDs:            []string{track.MbzArtistID},
				TrackMbzID:              track.MbzTrackID,
				ReleaseMbID:             track.MbzAlbumID,
			},
		},
	}
	return li
}

func (l *listenBrainzAgent) NowPlaying(ctx context.Context, userId string, track *model.MediaFile) error {
	sk, err := l.sessionKeys.Get(ctx, userId)
	if err != nil || sk == "" {
		return scrobbler.ErrNotAuthorized
	}

	li := l.formatListen(track)
	err = l.client.UpdateNowPlaying(ctx, sk, li)
	if err != nil {
		log.Warn(ctx, "ListenBrainz UpdateNowPlaying returned error", "track", track.Title, err)
		return scrobbler.ErrUnrecoverable
	}
	return nil
}

func (l *listenBrainzAgent) Scrobble(ctx context.Context, userId string, s scrobbler.Scrobble) error {
	sk, err := l.sessionKeys.Get(ctx, userId)
	if err != nil || sk == "" {
		return scrobbler.ErrNotAuthorized
	}

	li := l.formatListen(&s.MediaFile)
	li.ListenedAt = int(s.TimeStamp.Unix())
	err = l.client.Scrobble(ctx, sk, li)

	if err == nil {
		return nil
	}
	var lbErr *listenBrainzError
	isListenBrainzError := errors.As(err, &lbErr)
	if !isListenBrainzError {
		log.Warn(ctx, "ListenBrainz Scrobble returned HTTP error", "track", s.Title, err)
		return scrobbler.ErrRetryLater
	}
	if lbErr.Code == 500 || lbErr.Code == 503 {
		return scrobbler.ErrRetryLater
	}
	return scrobbler.ErrUnrecoverable
}

func (l *listenBrainzAgent) IsAuthorized(ctx context.Context, userId string) bool {
	sk, err := l.sessionKeys.Get(ctx, userId)
	return err == nil && sk != ""
}

func (l *listenBrainzAgent) CanProxyStars(ctx context.Context, userId string) bool {
	if !l.proxyStars {
		return false
	}
	return l.IsAuthorized(ctx, userId)
}

func (l *listenBrainzAgent) Star(ctx context.Context, userId string, star bool, tracks *model.MediaFiles) error {
	sk, err := l.sessionKeys.Get(ctx, userId)
	if err != nil || sk == "" {
		return scrobbler.ErrNotAuthorized
	}

	for _, track := range *tracks {
		if track.MbzTrackID != "" {
			err = l.client.Star(ctx, sk, star, track.MbzTrackID)
		}
	}

	return err
}

func init() {
	conf.AddHook(func() {
		if conf.Server.ListenBrainz.Enabled {
			scrobbler.Register(listenBrainzAgentName, func(ds model.DataStore) scrobbler.Scrobbler {
				return listenBrainzConstructor(ds)
			})
		}
	})
}

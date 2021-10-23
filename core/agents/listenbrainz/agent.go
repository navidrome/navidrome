package listenbrainz

import (
	"context"
	"net/http"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents/sessionkeys"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils"
)

const (
	listenBrainzAgentName = "listenbrainz"
	sessionKeyProperty    = "ListenBrainzSessionKey"
)

type listenBrainzAgent struct {
	ds          model.DataStore
	sessionKeys *sessionkeys.SessionKeys
	client      *Client
}

func listenBrainzConstructor(ds model.DataStore) *listenBrainzAgent {
	l := &listenBrainzAgent{
		ds:          ds,
		sessionKeys: &sessionkeys.SessionKeys{DataStore: ds, KeyName: sessionKeyProperty},
	}
	hc := &http.Client{
		Timeout: consts.DefaultHttpClientTimeOut,
	}
	chc := utils.NewCachedHTTPClient(hc, consts.DefaultHttpClientTimeOut)
	l.client = NewClient(chc)
	return l
}

func (l *listenBrainzAgent) AgentName() string {
	return listenBrainzAgentName
}

func (l *listenBrainzAgent) formatListen(ctx context.Context, track *model.MediaFile) listenInfo {
	player, _ := request.PlayerFrom(ctx)
	li := listenInfo{
		Track: trackMetadata{
			Artist: track.Artist,
			Track:  track.Title,
			Album:  track.Album,
			AdditionalInfo: additionalMetadata{
				TrackNumber: track.TrackNumber,
				MbzTrackID:  track.MbzTrackID,
				MbzAlbumID:  track.MbzAlbumID,
				Player:      player.Name,
			},
		},
	}
	if track.MbzArtistID != "" {
		li.Track.AdditionalInfo.MbzArtistIDs = []string{track.MbzArtistID}
	}
	return li
}

func (l *listenBrainzAgent) NowPlaying(ctx context.Context, userId string, track *model.MediaFile) error {
	sk, err := l.sessionKeys.Get(ctx, userId)
	if err != nil || sk == "" {
		return scrobbler.ErrNotAuthorized
	}

	li := l.formatListen(ctx, track)
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

	li := l.formatListen(ctx, &s.MediaFile)
	li.Timestamp = int(s.TimeStamp.Unix())
	err = l.client.Scrobble(ctx, sk, li)

	if err != nil {
		log.Warn(ctx, "ListenBrainz Scrobble returned error", "track", s.Title, err)
		return scrobbler.ErrUnrecoverable
	}
	return nil
}

func (l *listenBrainzAgent) IsAuthorized(ctx context.Context, userId string) bool {
	sk, err := l.sessionKeys.Get(ctx, userId)
	return err == nil && sk != ""
}

func init() {
	conf.AddHook(func() {
		if conf.Server.DevListenBrainzEnabled {
			scrobbler.Register(listenBrainzAgentName, func(ds model.DataStore) scrobbler.Scrobbler {
				return listenBrainzConstructor(ds)
			})
		}
	})
}

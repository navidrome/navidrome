package webhook

import (
	"context"
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
	webhookBaseAgentName   = "webhook-"
	sessionBaseKeyProperty = "WebHookSessionKey"
)

type webhookAgent struct {
	apiKey      string
	name        string
	sessionKeys *agents.SessionKeys
	url         string
	client      *client
}

func sessionKey(ds model.DataStore, name string) *agents.SessionKeys {
	return &agents.SessionKeys{
		DataStore: ds,
		KeyName:   sessionBaseKeyProperty + name,
	}
}

func webhookConstructor(ds model.DataStore, name, url, apiKey string) *webhookAgent {
	w := &webhookAgent{
		apiKey:      apiKey,
		name:        name,
		sessionKeys: sessionKey(ds, name),
		url:         url,
	}
	hc := &http.Client{
		Timeout: consts.DefaultHttpClientTimeOut,
	}
	chc := utils.NewCachedHTTPClient(hc, consts.DefaultHttpClientTimeOut)
	w.client = newClient(w.url, w.apiKey, chc)
	return w
}

func (w *webhookAgent) AgentName() string {
	return webhookBaseAgentName + w.name
}

func (w *webhookAgent) NowPlaying(ctx context.Context, userId string, track *model.MediaFile) error {
	sk, err := w.sessionKeys.Get(ctx, userId)
	if err != nil || sk == "" {
		return scrobbler.ErrNotAuthorized
	}

	err = w.client.scrobble(ctx, sk, false, ScrobbleInfo{
		artist:      track.Artist,
		track:       track.Title,
		album:       track.Album,
		trackNumber: track.TrackNumber,
		mbid:        track.MbzTrackID,
		duration:    int(track.Duration),
		albumArtist: track.AlbumArtist,
	})

	if err != nil {
		log.Warn(ctx, "Webhook client.updateNowPlaying returned error", "track", track.Title, "webhook", w.name, err)
		return scrobbler.ErrUnrecoverable
	}

	return nil
}

func (w *webhookAgent) Scrobble(ctx context.Context, userId string, s scrobbler.Scrobble) error {
	sk, err := w.sessionKeys.Get(ctx, userId)
	if err != nil || sk == "" {
		return scrobbler.ErrNotAuthorized
	}

	if s.Duration <= 30 {
		log.Debug(ctx, "Skipping webhook scrobble for short song", "track", s.Title, "duration", s.Duration, "webhook", w.name)
		return nil
	}

	err = w.client.scrobble(ctx, sk, true, ScrobbleInfo{
		artist:      s.Artist,
		track:       s.Title,
		album:       s.Album,
		trackNumber: s.TrackNumber,
		mbid:        s.MbzTrackID,
		duration:    int(s.Duration),
		albumArtist: s.AlbumArtist,
		timestamp:   s.TimeStamp,
	})

	if err != nil {
		log.Warn(ctx, "Webhook client.scrobble returned error", "track", s.Title, "webhook", w.name, err)
		return scrobbler.ErrUnrecoverable
	}

	return nil
}

func (w *webhookAgent) IsAuthorized(ctx context.Context, userId string) bool {
	sk, err := w.sessionKeys.Get(ctx, userId)
	return err == nil && sk != ""
}

func init() {
	conf.AddHook(func() {
		for _, webhook := range conf.Server.Webhooks {
			scrobbler.Register(webhookBaseAgentName+webhook.Name, func(ds model.DataStore) scrobbler.Scrobbler {
				return webhookConstructor(ds, webhook.Name, webhook.Url, webhook.ApiKey)
			})
		}
	})
}

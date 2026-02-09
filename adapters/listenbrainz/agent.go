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
	"github.com/navidrome/navidrome/utils/cache"
	"github.com/navidrome/navidrome/utils/slice"
)

const (
	listenBrainzAgentName = "listenbrainz"
	sessionKeyProperty    = "ListenBrainzSessionKey"
)

type listenBrainzAgent struct {
	ds          model.DataStore
	sessionKeys *agents.SessionKeys
	baseURL     string
	client      *client
}

func listenBrainzConstructor(ds model.DataStore) *listenBrainzAgent {
	l := &listenBrainzAgent{
		ds:          ds,
		sessionKeys: &agents.SessionKeys{DataStore: ds, KeyName: sessionKeyProperty},
		baseURL:     conf.Server.ListenBrainz.BaseURL,
	}
	hc := &http.Client{
		Timeout: consts.DefaultHttpClientTimeOut,
	}
	chc := cache.NewHTTPClient(hc, consts.DefaultHttpClientTimeOut)
	l.client = newClient(l.baseURL, chc)
	return l
}

func (l *listenBrainzAgent) AgentName() string {
	return listenBrainzAgentName
}

func (l *listenBrainzAgent) formatListen(track *model.MediaFile) listenInfo {
	artistMBIDs := slice.Map(track.Participants[model.RoleArtist], func(p model.Participant) string {
		return p.MbzArtistID
	})
	artistNames := slice.Map(track.Participants[model.RoleArtist], func(p model.Participant) string {
		return p.Name
	})
	li := listenInfo{
		TrackMetadata: trackMetadata{
			ArtistName:  track.Artist,
			TrackName:   track.Title,
			ReleaseName: track.Album,
			AdditionalInfo: additionalInfo{
				SubmissionClient:        consts.AppName,
				SubmissionClientVersion: consts.Version,
				TrackNumber:             track.TrackNumber,
				ArtistNames:             artistNames,
				ArtistMBIDs:             artistMBIDs,
				RecordingMBID:           track.MbzRecordingID,
				ReleaseMBID:             track.MbzAlbumID,
				ReleaseGroupMBID:        track.MbzReleaseGroupID,
				DurationMs:              int(track.Duration * 1000),
			},
		},
	}
	return li
}

func (l *listenBrainzAgent) NowPlaying(ctx context.Context, userId string, track *model.MediaFile, position int) error {
	sk, err := l.sessionKeys.Get(ctx, userId)
	if err != nil || sk == "" {
		return errors.Join(err, scrobbler.ErrNotAuthorized)
	}

	li := l.formatListen(track)
	err = l.client.updateNowPlaying(ctx, sk, li)
	if err != nil {
		log.Warn(ctx, "ListenBrainz updateNowPlaying returned error", "track", track.Title, err)
		return errors.Join(err, scrobbler.ErrUnrecoverable)
	}
	return nil
}

func (l *listenBrainzAgent) Scrobble(ctx context.Context, userId string, s scrobbler.Scrobble) error {
	sk, err := l.sessionKeys.Get(ctx, userId)
	if err != nil || sk == "" {
		return errors.Join(err, scrobbler.ErrNotAuthorized)
	}

	li := l.formatListen(&s.MediaFile)
	li.ListenedAt = int(s.TimeStamp.Unix())
	err = l.client.scrobble(ctx, sk, li)

	if err == nil {
		return nil
	}
	var lbErr *listenBrainzError
	isListenBrainzError := errors.As(err, &lbErr)
	if !isListenBrainzError {
		log.Warn(ctx, "ListenBrainz Scrobble returned HTTP error", "track", s.Title, err)
		return errors.Join(err, scrobbler.ErrRetryLater)
	}
	if lbErr.Code == 500 || lbErr.Code == 503 {
		return errors.Join(err, scrobbler.ErrRetryLater)
	}
	return errors.Join(err, scrobbler.ErrUnrecoverable)
}

func (l *listenBrainzAgent) IsAuthorized(ctx context.Context, userId string) bool {
	sk, err := l.sessionKeys.Get(ctx, userId)
	return err == nil && sk != ""
}

func (l *listenBrainzAgent) GetArtistURL(ctx context.Context, id, name, mbid string) (string, error) {
	if mbid == "" {
		return "", agents.ErrNotFound
	}

	url, err := l.client.getArtistUrl(ctx, mbid)
	if err != nil {
		return "", err
	}
	return url, nil
}

func (l *listenBrainzAgent) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error) {
	resp, err := l.client.getArtistTopSongs(ctx, mbid, count)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, agents.ErrNotFound
	}

	res := make([]agents.Song, len(resp))
	for i, t := range resp {
		mbid := ""
		if len(t.ArtistMBIDs) > 0 {
			mbid = t.ArtistMBIDs[0]
		}

		res[i] = agents.Song{
			Album:      t.ReleaseName,
			AlbumMBID:  t.ReleaseMBID,
			Artist:     t.ArtistName,
			ArtistMBID: mbid,
			Duration:   t.DurationMs,
			Name:       t.RecordingName,
			MBID:       t.RecordingMbid,
		}
	}
	return res, nil
}

func (l *listenBrainzAgent) GetSimilarArtists(ctx context.Context, id string, name string, mbid string, limit int) ([]agents.Artist, error) {
	if mbid == "" {
		return nil, agents.ErrNotFound
	}

	resp, err := l.client.getSimilarArtists(ctx, mbid, limit)
	if err != nil {
		return nil, err
	}

	if len(resp) == 0 {
		return nil, agents.ErrNotFound
	}

	artists := make([]agents.Artist, len(resp))
	for i, artist := range resp {
		artists[i] = agents.Artist{
			MBID: artist.MBID,
			Name: artist.Name,
		}
	}

	return artists, nil
}

func (l *listenBrainzAgent) GetSimilarSongsByTrack(ctx context.Context, id string, name string, artist string, mbid string, limit int) ([]agents.Song, error) {
	if mbid == "" {
		return nil, agents.ErrNotFound
	}

	resp, err := l.client.getSimilarRecordings(ctx, mbid, limit)
	if err != nil {
		return nil, err
	}

	if len(resp) == 0 {
		return nil, agents.ErrNotFound
	}

	songs := make([]agents.Song, len(resp))
	for i, song := range resp {
		songs[i] = agents.Song{
			Album:     song.ReleaseName,
			AlbumMBID: song.ReleaseMBID,
			Artist:    song.Artist,
			MBID:      song.MBID,
			Name:      song.Name,
		}
	}

	return songs, nil
}

func init() {
	conf.AddHook(func() {
		if conf.Server.ListenBrainz.Enabled {
			scrobbler.Register(listenBrainzAgentName, func(ds model.DataStore) scrobbler.Scrobbler {
				// This is a workaround for the fact that a (Interface)(nil) is not the same as a (*listenBrainzAgent)(nil)
				// See https://go.dev/doc/faq#nil_error
				a := listenBrainzConstructor(ds)
				if a != nil {
					return a
				}
				return nil
			})

			agents.Register(listenBrainzAgentName, func(ds model.DataStore) agents.Interface {
				// This is a workaround for the fact that a (Interface)(nil) is not the same as a (*listenBrainzAgent)(nil)
				// See https://go.dev/doc/faq#nil_error
				a := listenBrainzConstructor(ds)
				if a != nil {
					return a
				}
				return nil
			})
		}
	})
}

var (
	_ agents.ArtistTopSongsRetriever      = (*listenBrainzAgent)(nil)
	_ agents.ArtistURLRetriever           = (*listenBrainzAgent)(nil)
	_ agents.ArtistSimilarRetriever       = (*listenBrainzAgent)(nil)
	_ agents.SimilarSongsByTrackRetriever = (*listenBrainzAgent)(nil)
)

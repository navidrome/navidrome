package lastfm

import (
	"context"
	"net/http"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils"
)

const (
	lastFMAgentName = "lastfm"
	lastFMAPIKey    = "9b94a5515ea66b2da3ec03c12300327e"
	//lastFMAPISecret = "74cb6557cec7171d921af5d7d887c587" // Will be needed when implementing Scrobbling
)

type lastfmAgent struct {
	ctx    context.Context
	apiKey string
	lang   string
	client *Client
}

func lastFMConstructor(ctx context.Context) agents.Interface {
	l := &lastfmAgent{
		ctx:  ctx,
		lang: conf.Server.LastFM.Language,
	}
	if conf.Server.LastFM.ApiKey != "" {
		l.apiKey = conf.Server.LastFM.ApiKey
	} else {
		l.apiKey = lastFMAPIKey
	}
	hc := utils.NewCachedHTTPClient(http.DefaultClient, consts.DefaultCachedHttpClientTTL)
	l.client = NewClient(l.apiKey, l.lang, hc)
	return l
}

func (l *lastfmAgent) AgentName() string {
	return lastFMAgentName
}

func (l *lastfmAgent) GetMBID(id string, name string) (string, error) {
	a, err := l.callArtistGetInfo(name, "")
	if err != nil {
		return "", err
	}
	if a.MBID == "" {
		return "", agents.ErrNotFound
	}
	return a.MBID, nil
}

func (l *lastfmAgent) GetURL(id, name, mbid string) (string, error) {
	a, err := l.callArtistGetInfo(name, mbid)
	if err != nil {
		return "", err
	}
	if a.URL == "" {
		return "", agents.ErrNotFound
	}
	return a.URL, nil
}

func (l *lastfmAgent) GetBiography(id, name, mbid string) (string, error) {
	a, err := l.callArtistGetInfo(name, mbid)
	if err != nil {
		return "", err
	}
	if a.Bio.Summary == "" {
		return "", agents.ErrNotFound
	}
	return a.Bio.Summary, nil
}

func (l *lastfmAgent) GetSimilar(id, name, mbid string, limit int) ([]agents.Artist, error) {
	resp, err := l.callArtistGetSimilar(name, mbid, limit)
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

func (l *lastfmAgent) GetTopSongs(id, artistName, mbid string, count int) ([]agents.Song, error) {
	resp, err := l.callArtistGetTopTracks(artistName, mbid, count)
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

func (l *lastfmAgent) callArtistGetInfo(name string, mbid string) (*Artist, error) {
	a, err := l.client.ArtistGetInfo(l.ctx, name, mbid)
	lfErr, isLastFMError := err.(*lastFMError)
	if mbid != "" && ((err == nil && a.Name == "[unknown]") || (isLastFMError && lfErr.Code == 6)) {
		log.Warn(l.ctx, "LastFM/artist.getInfo could not find artist by mbid, trying again", "artist", name, "mbid", mbid)
		return l.callArtistGetInfo(name, "")
	}

	if err != nil {
		log.Error(l.ctx, "Error calling LastFM/artist.getInfo", "artist", name, "mbid", mbid, err)
		return nil, err
	}
	return a, nil
}

func (l *lastfmAgent) callArtistGetSimilar(name string, mbid string, limit int) ([]Artist, error) {
	s, err := l.client.ArtistGetSimilar(l.ctx, name, mbid, limit)
	lfErr, isLastFMError := err.(*lastFMError)
	if mbid != "" && ((err == nil && s.Attr.Artist == "[unknown]") || (isLastFMError && lfErr.Code == 6)) {
		log.Warn(l.ctx, "LastFM/artist.getSimilar could not find artist by mbid, trying again", "artist", name, "mbid", mbid)
		return l.callArtistGetSimilar(name, "", limit)
	}
	if err != nil {
		log.Error(l.ctx, "Error calling LastFM/artist.getSimilar", "artist", name, "mbid", mbid, err)
		return nil, err
	}
	return s.Artists, nil
}

func (l *lastfmAgent) callArtistGetTopTracks(artistName, mbid string, count int) ([]Track, error) {
	t, err := l.client.ArtistGetTopTracks(l.ctx, artistName, mbid, count)
	lfErr, isLastFMError := err.(*lastFMError)
	if mbid != "" && ((err == nil && t.Attr.Artist == "[unknown]") || (isLastFMError && lfErr.Code == 6)) {
		log.Warn(l.ctx, "LastFM/artist.getTopTracks could not find artist by mbid, trying again", "artist", artistName, "mbid", mbid)
		return l.callArtistGetTopTracks(artistName, "", count)
	}
	if err != nil {
		log.Error(l.ctx, "Error calling LastFM/artist.getTopTracks", "artist", artistName, "mbid", mbid, err)
		return nil, err
	}
	return t.Track, nil
}

func init() {
	conf.AddHook(func() {
		if conf.Server.LastFM.Enabled {
			agents.Register(lastFMAgentName, lastFMConstructor)
		}
	})
}

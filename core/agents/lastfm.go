package agents

import (
	"context"
	"net/http"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/lastfm"
	"github.com/navidrome/navidrome/log"
)

type lastfmAgent struct {
	ctx    context.Context
	apiKey string
	lang   string
	client *lastfm.Client
}

func lastFMConstructor(ctx context.Context) Interface {
	if conf.Server.LastFM.ApiKey == "" {
		return nil
	}

	l := &lastfmAgent{
		ctx:    ctx,
		apiKey: conf.Server.LastFM.ApiKey,
		lang:   conf.Server.LastFM.Language,
	}
	hc := NewCachedHTTPClient(http.DefaultClient, consts.DefaultCachedHttpClientTTL)
	l.client = lastfm.NewClient(l.apiKey, l.lang, hc)
	return l
}

func (l *lastfmAgent) AgentName() string {
	return "lastfm"
}

func (l *lastfmAgent) GetMBID(name string) (string, error) {
	a, err := l.callArtistGetInfo(name, "")
	if err != nil {
		return "", err
	}
	if a.MBID == "" {
		return "", ErrNotFound
	}
	return a.MBID, nil
}

func (l *lastfmAgent) GetURL(name, mbid string) (string, error) {
	a, err := l.callArtistGetInfo(name, mbid)
	if err != nil {
		return "", err
	}
	if a.URL == "" {
		return "", ErrNotFound
	}
	return a.URL, nil
}

func (l *lastfmAgent) GetBiography(name, mbid string) (string, error) {
	a, err := l.callArtistGetInfo(name, mbid)
	if err != nil {
		return "", err
	}
	if a.Bio.Summary == "" {
		return "", ErrNotFound
	}
	return a.Bio.Summary, nil
}

func (l *lastfmAgent) GetSimilar(name, mbid string, limit int) ([]Artist, error) {
	resp, err := l.callArtistGetSimilar(name, mbid, limit)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, ErrNotFound
	}
	var res []Artist
	for _, a := range resp {
		res = append(res, Artist{
			Name: a.Name,
			MBID: a.MBID,
		})
	}
	return res, nil
}

func (l *lastfmAgent) GetTopSongs(artistName, mbid string, count int) ([]Track, error) {
	resp, err := l.callArtistGetTopTracks(artistName, mbid, count)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, ErrNotFound
	}
	var res []Track
	for _, t := range resp {
		res = append(res, Track{
			Name: t.Name,
			MBID: t.MBID,
		})
	}
	return res, nil
}

func (l *lastfmAgent) callArtistGetInfo(name string, mbid string) (*lastfm.Artist, error) {
	a, err := l.client.ArtistGetInfo(l.ctx, name)
	if err != nil {
		log.Error(l.ctx, "Error calling LastFM/artist.getInfo", "artist", name, "mbid", mbid, err)
		return nil, err
	}
	return a, nil
}

func (l *lastfmAgent) callArtistGetSimilar(name string, mbid string, limit int) ([]lastfm.Artist, error) {
	s, err := l.client.ArtistGetSimilar(l.ctx, name, limit)
	if err != nil {
		log.Error(l.ctx, "Error calling LastFM/artist.getSimilar", "artist", name, "mbid", mbid, err)
		return nil, err
	}
	return s, nil
}

func (l *lastfmAgent) callArtistGetTopTracks(artistName, mbid string, count int) ([]lastfm.Track, error) {
	t, err := l.client.ArtistGetTopTracks(l.ctx, artistName, count)
	if err != nil {
		log.Error(l.ctx, "Error calling LastFM/artist.getTopTracks", "artist", artistName, "mbid", mbid, err)
		return nil, err
	}
	return t, nil
}

func init() {
	conf.AddHook(func() {
		if conf.Server.LastFM.ApiKey != "" {
			log.Info("Last.FM integration is ENABLED")
			Register("lastfm", lastFMConstructor)
		}
	})
}

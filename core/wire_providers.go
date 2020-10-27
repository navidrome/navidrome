package core

import (
	"net/http"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/core/lastfm"
	"github.com/deluan/navidrome/core/spotify"
	"github.com/deluan/navidrome/core/transcoder"
	"github.com/google/wire"
)

var Set = wire.NewSet(
	NewArtwork,
	NewMediaStreamer,
	NewTranscodingCache,
	NewImageCache,
	NewArchiver,
	NewNowPlayingRepository,
	NewExternalInfo,
	NewCacheWarmer,
	LastFMNewClient,
	SpotifyNewClient,
	transcoder.New,
)

func LastFMNewClient() *lastfm.Client {
	if conf.Server.LastFM.ApiKey == "" {
		return nil
	}

	return lastfm.NewClient(conf.Server.LastFM.ApiKey, conf.Server.LastFM.Language, http.DefaultClient)
}

func SpotifyNewClient() *spotify.Client {
	if conf.Server.Spotify.ID == "" || conf.Server.Spotify.Secret == "" {
		return nil
	}

	return spotify.NewClient(conf.Server.Spotify.ID, conf.Server.Spotify.Secret, http.DefaultClient)
}

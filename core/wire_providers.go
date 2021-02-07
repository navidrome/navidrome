package core

import (
	"net/http"

	"github.com/google/wire"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/lastfm"
	"github.com/navidrome/navidrome/core/spotify"
	"github.com/navidrome/navidrome/core/transcoder"
)

var Set = wire.NewSet(
	NewArtwork,
	NewMediaStreamer,
	GetTranscodingCache,
	GetImageCache,
	NewArchiver,
	NewNowPlayingRepository,
	NewExternalInfo,
	NewCacheWarmer,
	NewPlayers,
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

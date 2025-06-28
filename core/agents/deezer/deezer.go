package deezer

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/cache"
)

const deezerAgentName = "deezer"
const deezerApiPictureXlSize = 1000
const deezerApiPictureBigSize = 500
const deezerApiPictureMediumSize = 250
const deezerApiPictureSmallSize = 56
const deezerArtistSearchLimit = 50

type deezerAgent struct {
	dataStore model.DataStore
	client    *client
}

func deezerConstructor(dataStore model.DataStore) agents.Interface {
	agent := &deezerAgent{dataStore: dataStore}
	httpClient := &http.Client{
		Timeout: consts.DefaultHttpClientTimeOut,
	}
	cachedHttpClient := cache.NewHTTPClient(httpClient, consts.DefaultHttpClientTimeOut)
	agent.client = newClient(cachedHttpClient)
	return agent
}

func (s *deezerAgent) AgentName() string {
	return deezerAgentName
}

func (s *deezerAgent) GetArtistImages(ctx context.Context, _, name, _ string) ([]agents.ExternalImage, error) {
	artist, err := s.searchArtist(ctx, name)
	if err != nil {
		if errors.Is(err, agents.ErrNotFound) {
			log.Warn(ctx, "Artist not found in deezer", "artist", name)
		} else {
			log.Error(ctx, "Error calling deezer", "artist", name, err)
		}
		return nil, err
	}

	var res []agents.ExternalImage
	possibleImages := []struct {
		URL  string
		Size int
	}{
		{artist.PictureXl, deezerApiPictureXlSize},
		{artist.PictureBig, deezerApiPictureBigSize},
		{artist.PictureMedium, deezerApiPictureMediumSize},
		{artist.PictureSmall, deezerApiPictureSmallSize},
	}
	for _, imgData := range possibleImages {
		if imgData.URL != "" {
			res = append(res, agents.ExternalImage{
				URL:  imgData.URL,
				Size: imgData.Size,
			})
		}
	}
	return res, nil
}

func (s *deezerAgent) searchArtist(ctx context.Context, name string) (*Artist, error) {
	artists, err := s.client.searchArtists(ctx, name, deezerArtistSearchLimit)
	if errors.Is(err, ErrNotFound) || len(artists) == 0 {
		return nil, agents.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// If the first one has the same name, that's the one
	if !strings.EqualFold(artists[0].Name, name) {
		return nil, agents.ErrNotFound
	}
	return &artists[0], err
}

func init() {
	conf.AddHook(func() {
		if conf.Server.Deezer.Enabled {
			agents.Register(deezerAgentName, deezerConstructor)
		}
	})
}

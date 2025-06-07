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

type deezerAgent struct {
	dataStore model.DataStore
	client    *client
}

func deezerConstructor(dataStore model.DataStore) agents.Interface {
	agent := &deezerAgent{dataStore: dataStore}
	httpClient := &http.Client{
		Timeout: consts.DefaultHttpClientTimeOut,
	}
	cached_http_client := cache.NewHTTPClient(httpClient, consts.DefaultHttpClientTimeOut)
	agent.client = newClient(cached_http_client)
	return agent
}

func (s *deezerAgent) AgentName() string {
	return deezerAgentName
}

func (s *deezerAgent) GetArtistImages(ctx context.Context, id, name, mbid string) ([]agents.ExternalImage, error) {
	artist, err := s.searchArtist(ctx, name)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			log.Warn(ctx, "Artist not found in deezer", "artist", name)
		} else {
			log.Error(ctx, "Error calling deezer", "artist", name, err)
		}
		return nil, err
	}

	var res []agents.ExternalImage
	res = append(res, agents.ExternalImage{
		URL:  artist.PictureXl,
		Size: 1000,
	})
	res = append(res, agents.ExternalImage{
		URL:  artist.PictureBig,
		Size: 500,
	})
	res = append(res, agents.ExternalImage{
		URL:  artist.PictureMedium,
		Size: 250,
	})
	res = append(res, agents.ExternalImage{
		URL:  artist.PictureSmall,
		Size: 56,
	})
	return res, nil
}

func (s *deezerAgent) searchArtist(ctx context.Context, name string) (*Artist, error) {
	artists, err := s.client.searchArtists(ctx, name, 50)
	if err != nil || len(artists) == 0 {
		return nil, model.ErrNotFound
	}

	// If the first one has the same name, that's the one
	if !strings.EqualFold(artists[0].Name, name) {
		return nil, model.ErrNotFound
	}
	return &artists[0], err
}

func init() {
	conf.AddHook(func() {
		agents.Register(deezerAgentName, deezerConstructor)
	})
}

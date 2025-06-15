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
const DEEZER_API_PICTURE_XL_SIZE = 1000
const DEEZER_API_PICTURE_BIG_SIZE = 500
const DEEZER_API_PICTURE_MEDIUM_SIZE = 250
const DEEZER_API_PICTURE_SMALL_SIZE = 56

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

func (s *deezerAgent) GetArtistImages(ctx context.Context, id, name, mbid string) ([]agents.ExternalImage, error) {
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
	res = append(res, agents.ExternalImage{
		URL:  artist.PictureXl,
		Size: DEEZER_API_PICTURE_XL_SIZE,
	})
	res = append(res, agents.ExternalImage{
		URL:  artist.PictureBig,
		Size: DEEZER_API_PICTURE_BIG_SIZE,
	})
	res = append(res, agents.ExternalImage{
		URL:  artist.PictureMedium,
		Size: DEEZER_API_PICTURE_MEDIUM_SIZE,
	})
	res = append(res, agents.ExternalImage{
		URL:  artist.PictureSmall,
		Size: DEEZER_API_PICTURE_SMALL_SIZE,
	})
	return res, nil
}

func (s *deezerAgent) searchArtist(ctx context.Context, name string) (*Artist, error) {
	artists, err := s.client.searchArtists(ctx, name, 50)
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
		agents.Register(deezerAgentName, deezerConstructor)
	})
}

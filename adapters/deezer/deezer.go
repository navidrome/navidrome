package deezer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/cache"
	"github.com/navidrome/navidrome/utils/slice"
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
	agent.client = newClient(cachedHttpClient, conf.Server.Deezer.Language)
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

	log.Trace(ctx, "Artists found", "count", len(artists), "searched_name", name)
	for i := range artists {
		log.Trace(ctx, fmt.Sprintf("Artists found #%d", i), "name", artists[i].Name, "id", artists[i].ID, "link", artists[i].Link)
		if i > 2 {
			break
		}
	}

	// If the first one has the same name, that's the one
	if !strings.EqualFold(artists[0].Name, name) {
		log.Trace(ctx, "Top artist do not match", "searched_name", name, "found_name", artists[0].Name)
		return nil, agents.ErrNotFound
	}
	log.Trace(ctx, "Found artist", "name", artists[0].Name, "id", artists[0].ID, "link", artists[0].Link)
	return &artists[0], err
}

func (s *deezerAgent) GetSimilarArtists(ctx context.Context, _, name, _ string, limit int) ([]agents.Artist, error) {
	artist, err := s.searchArtist(ctx, name)
	if err != nil {
		return nil, err
	}

	related, err := s.client.getRelatedArtists(ctx, artist.ID)
	if err != nil {
		return nil, err
	}

	res := slice.Map(related, func(r Artist) agents.Artist {
		return agents.Artist{
			Name: r.Name,
		}
	})
	if len(res) > limit {
		res = res[:limit]
	}
	return res, nil
}

func (s *deezerAgent) GetArtistTopSongs(ctx context.Context, _, artistName, _ string, count int) ([]agents.Song, error) {
	artist, err := s.searchArtist(ctx, artistName)
	if err != nil {
		return nil, err
	}

	tracks, err := s.client.getTopTracks(ctx, artist.ID, count)
	if err != nil {
		return nil, err
	}

	res := slice.Map(tracks, func(r Track) agents.Song {
		return agents.Song{
			Name:  r.Title,
			Album: r.Album.Title,
		}
	})
	return res, nil
}

func (s *deezerAgent) GetArtistBiography(ctx context.Context, _, name, _ string) (string, error) {
	artist, err := s.searchArtist(ctx, name)
	if err != nil {
		return "", err
	}

	return s.client.getArtistBio(ctx, artist.ID)
}

func init() {
	conf.AddHook(func() {
		if conf.Server.Deezer.Enabled {
			agents.Register(deezerAgentName, deezerConstructor)
		}
	})
}

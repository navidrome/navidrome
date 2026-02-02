package tidal

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
	"github.com/navidrome/navidrome/utils/slice"
)

const tidalAgentName = "tidal"
const tidalArtistSearchLimit = 20

type tidalAgent struct {
	ds     model.DataStore
	client *client
}

func tidalConstructor(ds model.DataStore) agents.Interface {
	if conf.Server.Tidal.ClientID == "" || conf.Server.Tidal.ClientSecret == "" {
		return nil
	}
	l := &tidalAgent{
		ds: ds,
	}
	hc := &http.Client{
		Timeout: consts.DefaultHttpClientTimeOut,
	}
	chc := cache.NewHTTPClient(hc, consts.DefaultHttpClientTimeOut)
	l.client = newClient(conf.Server.Tidal.ClientID, conf.Server.Tidal.ClientSecret, chc)
	return l
}

func (t *tidalAgent) AgentName() string {
	return tidalAgentName
}

func (t *tidalAgent) GetArtistImages(ctx context.Context, id, name, mbid string) ([]agents.ExternalImage, error) {
	artist, err := t.searchArtist(ctx, name)
	if err != nil {
		if errors.Is(err, agents.ErrNotFound) {
			log.Warn(ctx, "Artist not found in Tidal", "artist", name)
		} else {
			log.Error(ctx, "Error calling Tidal", "artist", name, err)
		}
		return nil, err
	}

	var res []agents.ExternalImage
	for _, img := range artist.Attributes.Picture {
		res = append(res, agents.ExternalImage{
			URL:  img.URL,
			Size: img.Width,
		})
	}

	// Sort images by size descending
	if len(res) == 0 {
		return nil, agents.ErrNotFound
	}

	return res, nil
}

func (t *tidalAgent) GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]agents.Artist, error) {
	artist, err := t.searchArtist(ctx, name)
	if err != nil {
		return nil, err
	}

	similar, err := t.client.getSimilarArtists(ctx, artist.ID, limit)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, agents.ErrNotFound
		}
		return nil, err
	}

	res := slice.Map(similar, func(a ArtistResource) agents.Artist {
		return agents.Artist{
			Name: a.Attributes.Name,
		}
	})

	return res, nil
}

func (t *tidalAgent) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error) {
	artist, err := t.searchArtist(ctx, artistName)
	if err != nil {
		return nil, err
	}

	tracks, err := t.client.getArtistTopTracks(ctx, artist.ID, count)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, agents.ErrNotFound
		}
		return nil, err
	}

	res := slice.Map(tracks, func(track TrackResource) agents.Song {
		return agents.Song{
			Name:     track.Attributes.Title,
			ISRC:     track.Attributes.ISRC,
			Duration: uint32(track.Attributes.Duration * 1000), // Convert seconds to milliseconds
		}
	})

	return res, nil
}

func (t *tidalAgent) searchArtist(ctx context.Context, name string) (*ArtistResource, error) {
	artists, err := t.client.searchArtists(ctx, name, tidalArtistSearchLimit)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, agents.ErrNotFound
		}
		return nil, err
	}

	if len(artists) == 0 {
		return nil, agents.ErrNotFound
	}

	// Find exact match (case-insensitive)
	for i := range artists {
		if strings.EqualFold(artists[i].Attributes.Name, name) {
			log.Trace(ctx, "Found artist in Tidal", "name", artists[i].Attributes.Name, "id", artists[i].ID)
			return &artists[i], nil
		}
	}

	// If no exact match, check if first result is close enough
	log.Trace(ctx, "No exact artist match in Tidal", "searched", name, "found", artists[0].Attributes.Name)
	return nil, agents.ErrNotFound
}

func init() {
	conf.AddHook(func() {
		if conf.Server.Tidal.Enabled {
			agents.Register(tidalAgentName, tidalConstructor)
		}
	})
}

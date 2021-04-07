package agents

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/spotify"
	"github.com/xrash/smetrics"
)

const spotifyAgentName = "spotify"

type spotifyAgent struct {
	ctx    context.Context
	id     string
	secret string
	client *spotify.Client
}

func spotifyConstructor(ctx context.Context) Interface {
	l := &spotifyAgent{
		ctx:    ctx,
		id:     conf.Server.Spotify.ID,
		secret: conf.Server.Spotify.Secret,
	}
	hc := NewCachedHTTPClient(http.DefaultClient, consts.DefaultCachedHttpClientTTL)
	l.client = spotify.NewClient(l.id, l.secret, hc)
	return l
}

func (s *spotifyAgent) AgentName() string {
	return spotifyAgentName
}

func (s *spotifyAgent) GetImages(id, name, mbid string) ([]ArtistImage, error) {
	a, err := s.searchArtist(name)
	if err != nil {
		if err == model.ErrNotFound {
			log.Warn(s.ctx, "Artist not found in Spotify", "artist", name)
		} else {
			log.Error(s.ctx, "Error calling Spotify", "artist", name, err)
		}
		return nil, err
	}

	var res []ArtistImage
	for _, img := range a.Images {
		res = append(res, ArtistImage{
			URL:  img.URL,
			Size: img.Width,
		})
	}
	return res, nil
}

func (s *spotifyAgent) searchArtist(name string) (*spotify.Artist, error) {
	artists, err := s.client.SearchArtists(s.ctx, name, 40)
	if err != nil || len(artists) == 0 {
		return nil, model.ErrNotFound
	}
	name = strings.ToLower(name)

	// Sort results, prioritizing artists with images, with similar names and with high popularity, in this order
	sort.Slice(artists, func(i, j int) bool {
		ai := fmt.Sprintf("%-5t-%03d-%04d", len(artists[i].Images) == 0, smetrics.WagnerFischer(name, strings.ToLower(artists[i].Name), 1, 1, 2), 1000-artists[i].Popularity)
		aj := fmt.Sprintf("%-5t-%03d-%04d", len(artists[j].Images) == 0, smetrics.WagnerFischer(name, strings.ToLower(artists[j].Name), 1, 1, 2), 1000-artists[j].Popularity)
		return ai < aj
	})

	// If the first one has the same name, that's the one
	if strings.ToLower(artists[0].Name) != name {
		return nil, model.ErrNotFound
	}
	return &artists[0], err
}

func init() {
	conf.AddHook(func() {
		if conf.Server.Spotify.ID != "" && conf.Server.Spotify.Secret != "" {
			log.Info("Spotify integration is ENABLED")
			Register(spotifyAgentName, spotifyConstructor)
		}
	})
}

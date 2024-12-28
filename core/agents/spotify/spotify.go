package spotify

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/cache"
	"github.com/xrash/smetrics"
)

const spotifyAgentName = "spotify"

type spotifyAgent struct {
	ds     model.DataStore
	id     string
	secret string
	client *client
}

func spotifyConstructor(ds model.DataStore) agents.Interface {
	l := &spotifyAgent{
		ds:     ds,
		id:     conf.Server.Spotify.ID,
		secret: conf.Server.Spotify.Secret,
	}
	hc := &http.Client{
		Timeout: consts.DefaultHttpClientTimeOut,
	}
	chc := cache.NewHTTPClient(hc, consts.DefaultHttpClientTimeOut)
	l.client = newClient(l.id, l.secret, chc)
	return l
}

func (s *spotifyAgent) AgentName() string {
	return spotifyAgentName
}

func (s *spotifyAgent) GetArtistImages(ctx context.Context, id, name, sortName, mbid string) ([]agents.ExternalImage, error) {
	a, err := s.searchArtist(ctx, name, sortName)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			log.Warn(ctx, "Artist not found in Spotify", "artist", name)
		} else {
			log.Error(ctx, "Error calling Spotify", "artist", name, err)
		}
		return nil, err
	}

	var res []agents.ExternalImage
	for _, img := range a.Images {
		res = append(res, agents.ExternalImage{
			URL:  img.URL,
			Size: img.Width,
		})
	}
	return res, nil
}

func (s *spotifyAgent) searchArtist(ctx context.Context, name string, sortName string) (*Artist, error) {
	artists, err := s.client.searchArtists(ctx, name, 40)
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

	// If the first result sorted by name matches the name, return that artist (should cover most use cases with English names)
	if strings.ToLower(artists[0].Name) == name {
		return &artists[0], err
	}

	// If matching by name doesn't work(non-Latin alphabet names), match by sortName
	sortName = strings.ToLower(sortName)
	sort.Slice(artists, func(i, j int) bool {
		ai := fmt.Sprintf("%-5t-%03d-%04d", len(artists[i].Images) == 0, customMetric(strings.ToLower(artists[i].Name), sortName), 1000-artists[i].Popularity)
		aj := fmt.Sprintf("%-5t-%03d-%04d", len(artists[j].Images) == 0, customMetric(strings.ToLower(artists[j].Name), sortName), 1000-artists[j].Popularity)
		return ai < aj
	})

	// If the first one has a similar name to sortName, that is the one
	if customMetric(strings.ToLower(artists[0].Name), sortName) < 1 {
		return &artists[0], err
	}
	return nil, model.ErrNotFound
}

func customMetric(name string, sortName string) int {
	// Because sortNames often have commas and have different word orders(e.g. "John Williams" -> "Williams, John", "The Beatles"->"Beatles, The"),
	// this custom metric returns the number of letters that are different from "name" and "sortName" but irrespective of the word order
	name = strings.ReplaceAll(name, " ", "")
	sortName = strings.ReplaceAll(sortName, " ", "")
	substrings := strings.Split(sortName, ",")
	for _, ss := range substrings {
		name = strings.Replace(name, ss, "", 1)
	}
	return len(name)
}

func init() {
	conf.AddHook(func() {
		if conf.Server.Spotify.ID != "" && conf.Server.Spotify.Secret != "" {
			agents.Register(spotifyAgentName, spotifyConstructor)
		}
	})
}

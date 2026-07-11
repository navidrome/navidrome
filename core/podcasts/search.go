package podcasts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// FeedSearchResult is a normalized podcast search hit, independent of the
// upstream directory API used to produce it.
type FeedSearchResult struct {
	Title        string `json:"title"`
	Author       string `json:"author,omitempty"`
	FeedUrl      string `json:"feedUrl"`
	ArtworkUrl   string `json:"artworkUrl,omitempty"`
	EpisodeCount int    `json:"episodeCount,omitempty"`
}

const searchResultLimit = 25

// iTunes Search API response shapes. Field names confirmed against a live
// request to https://itunes.apple.com/search?media=podcast&entity=podcast.
type itunesSearchResponse struct {
	Results []itunesSearchResult `json:"results"`
}

type itunesSearchResult struct {
	CollectionName string `json:"collectionName"`
	ArtistName     string `json:"artistName"`
	FeedUrl        string `json:"feedUrl"`
	ArtworkUrl600  string `json:"artworkUrl600"`
	ArtworkUrl100  string `json:"artworkUrl100"`
	TrackCount     int    `json:"trackCount"`
}

func searchFeeds(ctx context.Context, query string) ([]FeedSearchResult, error) {
	u := "https://itunes.apple.com/search?" + url.Values{
		"term":   {query},
		"media":  {"podcast"},
		"entity": {"podcast"},
		"limit":  {fmt.Sprint(searchResultLimit)},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("searching podcast directory: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("podcast directory search returned status %d", resp.StatusCode)
	}

	var parsed itunesSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("parsing podcast directory response: %w", err)
	}

	results := make([]FeedSearchResult, 0, len(parsed.Results))
	for _, r := range parsed.Results {
		if r.FeedUrl == "" {
			continue
		}
		artwork := r.ArtworkUrl600
		if artwork == "" {
			artwork = r.ArtworkUrl100
		}
		results = append(results, FeedSearchResult{
			Title:        r.CollectionName,
			Author:       r.ArtistName,
			FeedUrl:      r.FeedUrl,
			ArtworkUrl:   artwork,
			EpisodeCount: r.TrackCount,
		})
	}
	return results, nil
}

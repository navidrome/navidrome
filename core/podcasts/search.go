package podcasts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
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

const (
	searchResultLimit = 25
	topFeedsLimit     = 10
	defaultTopCountry = "us"
)

var countryCodePattern = regexp.MustCompile(`^[a-zA-Z]{2}$`)

// iTunes Search/Lookup API response shapes. Field names confirmed against
// live requests to https://itunes.apple.com/search and .../lookup.
type itunesLookupResponse struct {
	Results []itunesResult `json:"results"`
}

type itunesResult struct {
	CollectionId   int    `json:"collectionId"`
	CollectionName string `json:"collectionName"`
	ArtistName     string `json:"artistName"`
	FeedUrl        string `json:"feedUrl"`
	ArtworkUrl600  string `json:"artworkUrl600"`
	ArtworkUrl100  string `json:"artworkUrl100"`
	TrackCount     int    `json:"trackCount"`
}

func (r itunesResult) toFeedSearchResult() FeedSearchResult {
	artwork := r.ArtworkUrl600
	if artwork == "" {
		artwork = r.ArtworkUrl100
	}
	return FeedSearchResult{
		Title:        r.CollectionName,
		Author:       r.ArtistName,
		FeedUrl:      r.FeedUrl,
		ArtworkUrl:   artwork,
		EpisodeCount: r.TrackCount,
	}
}

func searchFeeds(ctx context.Context, query string) ([]FeedSearchResult, error) {
	u := "https://itunes.apple.com/search?" + url.Values{
		"term":   {query},
		"media":  {"podcast"},
		"entity": {"podcast"},
		"limit":  {strconv.Itoa(searchResultLimit)},
	}.Encode()

	var parsed itunesLookupResponse
	if err := getJSON(ctx, u, &parsed); err != nil {
		return nil, fmt.Errorf("searching podcast directory: %w", err)
	}

	results := make([]FeedSearchResult, 0, len(parsed.Results))
	for _, r := range parsed.Results {
		if r.FeedUrl == "" {
			continue
		}
		results = append(results, r.toFeedSearchResult())
	}
	return results, nil
}

// topFeeds returns the current top podcast chart for the given ISO 3166-1
// alpha-2 country code (e.g. "us", "au", "gb"), sourced from Apple's public
// Marketing Tools API and resolved to real RSS feed URLs via a bulk iTunes
// Lookup call. Falls back to the US chart for an invalid/unknown country.
func topFeeds(ctx context.Context, country string) ([]FeedSearchResult, error) {
	country = strings.ToLower(strings.TrimSpace(country))
	if !countryCodePattern.MatchString(country) {
		country = defaultTopCountry
	}

	chartUrl := fmt.Sprintf("https://rss.marketingtools.apple.com/api/v2/%s/podcasts/top/%d/podcasts.json", country, topFeedsLimit)
	var chart struct {
		Feed struct {
			Results []struct {
				ID string `json:"id"`
			} `json:"results"`
		} `json:"feed"`
	}
	if err := getJSON(ctx, chartUrl, &chart); err != nil {
		return nil, fmt.Errorf("fetching top podcasts chart: %w", err)
	}
	if len(chart.Feed.Results) == 0 {
		return nil, nil
	}

	ids := make([]string, 0, len(chart.Feed.Results))
	order := make(map[string]int, len(chart.Feed.Results))
	for i, r := range chart.Feed.Results {
		ids = append(ids, r.ID)
		order[r.ID] = i
	}

	lookupUrl := "https://itunes.apple.com/lookup?" + url.Values{
		"id":     {strings.Join(ids, ",")},
		"entity": {"podcast"},
	}.Encode()
	var lookup itunesLookupResponse
	if err := getJSON(ctx, lookupUrl, &lookup); err != nil {
		return nil, fmt.Errorf("resolving top podcasts feed URLs: %w", err)
	}

	results := make([]FeedSearchResult, len(chart.Feed.Results))
	found := make([]bool, len(chart.Feed.Results))
	for _, r := range lookup.Results {
		if r.FeedUrl == "" {
			continue
		}
		idx, ok := order[strconv.Itoa(r.CollectionId)]
		if !ok {
			continue
		}
		results[idx] = r.toFeedSearchResult()
		found[idx] = true
	}

	ordered := make([]FeedSearchResult, 0, len(results))
	for i, ok := range found {
		if ok {
			ordered = append(ordered, results[i])
		}
	}
	return ordered, nil
}

func getJSON(ctx context.Context, u string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, u)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("parsing response from %s: %w", u, err)
	}
	return nil
}

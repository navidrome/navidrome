package deezer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/navidrome/navidrome/log"
)

const apiBaseURL = "https://api.deezer.com"

var (
	ErrNotFound = errors.New("deezer: not found")
)

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type client struct {
	httpDoer httpDoer
}

func newClient(hc httpDoer) *client {
	return &client{hc}
}

func (c *client) searchArtists(ctx context.Context, name string, limit int) ([]Artist, error) {
	params := url.Values{}
	params.Add("q", name)
	params.Add("limit", strconv.Itoa(limit))
	req, err := http.NewRequestWithContext(ctx, "GET", apiBaseURL+"/search/artist", nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = params.Encode()

	var results SearchArtistResults
	err = c.makeRequest(req, &results)
	if err != nil {
		return nil, err
	}

	if len(results.Data) == 0 {
		return nil, ErrNotFound
	}
	return results.Data, nil
}

func (c *client) makeRequest(req *http.Request, response interface{}) error {
	log.Trace(req.Context(), fmt.Sprintf("Sending Deezer %s request", req.Method), "url", req.URL)
	resp, err := c.httpDoer.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return c.parseError(data)
	}

	return json.Unmarshal(data, response)
}

func (c *client) parseError(data []byte) error {
	var deezerError Error
	err := json.Unmarshal(data, &deezerError)
	if err != nil {
		return err
	}
	return fmt.Errorf("deezer error(%d): %s", deezerError.Error.Code, deezerError.Error.Message)
}

func (c *client) getRelatedArtists(ctx context.Context, artistID int) ([]Artist, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/artist/%d/related", apiBaseURL, artistID), nil)
	if err != nil {
		return nil, err
	}

	var results RelatedArtists
	err = c.makeRequest(req, &results)
	if err != nil {
		return nil, err
	}

	return results.Data, nil
}

func (c *client) getTopTracks(ctx context.Context, artistID int, limit int) ([]Track, error) {
	params := url.Values{}
	params.Add("limit", strconv.Itoa(limit))
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/artist/%d/top", apiBaseURL, artistID), nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = params.Encode()

	var results TopTracks
	err = c.makeRequest(req, &results)
	if err != nil {
		return nil, err
	}

	return results.Data, nil
}

var dzrAppStateRegex = regexp.MustCompile(`window\.__DZR_APP_STATE__\s*=\s*({.+?})\s*</script>`)
var strictPolicy = bluemonday.StrictPolicy()

func (c *client) getArtistBio(ctx context.Context, artistID int) (string, error) {
	u := fmt.Sprintf("https://www.deezer.com/en/artist/%d/biography", artistID)
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return "", err
	}

	log.Trace(ctx, "Fetching Deezer artist biography", "url", u)
	resp, err := c.httpDoer.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("deezer: failed to fetch biography: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	matches := dzrAppStateRegex.FindSubmatch(body)
	if len(matches) < 2 {
		return "", errors.New("deezer: could not find __DZR_APP_STATE__")
	}

	type appState struct {
		Bio struct {
			Bio    string `json:"BIO"`
			Resume string `json:"RESUME"`
		} `json:"BIO"`
	}

	var state appState
	if err := json.Unmarshal(matches[1], &state); err != nil {
		return "", fmt.Errorf("deezer: failed to parse __DZR_APP_STATE__: %w", err)
	}

	var bio string
	if state.Bio.Bio != "" {
		bio = state.Bio.Bio
	} else {
		bio = state.Bio.Resume
	}

	return cleanBio(bio), nil
}

func cleanBio(bio string) string {
	bio = strings.ReplaceAll(bio, "</p>", "\n")
	return strictPolicy.Sanitize(bio)
}

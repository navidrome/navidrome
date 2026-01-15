package deezer

import (
	bytes "bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/navidrome/navidrome/log"
)

const apiBaseURL = "https://api.deezer.com"
const authBaseURL = "https://auth.deezer.com"

var (
	ErrNotFound = errors.New("deezer: not found")
)

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type client struct {
	httpDoer httpDoer
	language string
	jwt      jwtToken
}

func newClient(hc httpDoer, language string) *client {
	return &client{
		httpDoer: hc,
		language: language,
	}
}

func (c *client) searchArtists(ctx context.Context, name string, limit int) ([]Artist, error) {
	params := url.Values{}
	params.Add("q", name)
	params.Add("order", "RANKING")
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

func (c *client) makeRequest(req *http.Request, response any) error {
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

const pipeAPIURL = "https://pipe.deezer.com/api"

var strictPolicy = bluemonday.StrictPolicy()

func (c *client) getArtistBio(ctx context.Context, artistID int) (string, error) {
	jwt, err := c.getJWT(ctx)
	if err != nil {
		return "", fmt.Errorf("deezer: failed to get JWT: %w", err)
	}

	query := map[string]any{
		"operationName": "ArtistBio",
		"variables": map[string]any{
			"artistId": strconv.Itoa(artistID),
		},
		"query": `query ArtistBio($artistId: String!) {
			artist(artistId: $artistId) {
				bio {
					full
				}
			}
		}`,
	}

	body, err := json.Marshal(query)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", pipeAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", c.language)
	req.Header.Set("Authorization", "Bearer "+jwt)

	log.Trace(ctx, "Fetching Deezer artist biography via GraphQL", "artistId", artistID, "language", c.language)
	resp, err := c.httpDoer.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("deezer: failed to fetch biography: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	type graphQLResponse struct {
		Data struct {
			Artist struct {
				Bio struct {
					Full string `json:"full"`
				} `json:"bio"`
			} `json:"artist"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		}
	}

	var result graphQLResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("deezer: failed to parse GraphQL response: %w", err)
	}

	if len(result.Errors) > 0 {
		var errs []error
		for m := range result.Errors {
			errs = append(errs, errors.New(result.Errors[m].Message))
		}
		err := errors.Join(errs...)
		return "", fmt.Errorf("deezer: GraphQL error: %w", err)
	}

	if result.Data.Artist.Bio.Full == "" {
		return "", errors.New("deezer: biography not found")
	}

	return cleanBio(result.Data.Artist.Bio.Full), nil
}

func cleanBio(bio string) string {
	bio = strings.ReplaceAll(bio, "</p>", "\n")
	return strictPolicy.Sanitize(bio)
}

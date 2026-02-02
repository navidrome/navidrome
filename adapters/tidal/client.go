package tidal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/navidrome/navidrome/log"
)

const (
	apiBaseURL   = "https://openapi.tidal.com"
	authTokenURL = "https://auth.tidal.com/v1/oauth2/token"
)

var (
	ErrNotFound = errors.New("tidal: not found")
)

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type client struct {
	id         string
	secret     string
	hc         httpDoer
	token      string
	tokenExp   time.Time
	tokenMutex sync.Mutex
}

func newClient(id, secret string, hc httpDoer) *client {
	return &client{
		id:     id,
		secret: secret,
		hc:     hc,
	}
}

func (c *client) searchArtists(ctx context.Context, name string, limit int) ([]ArtistResource, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	params := url.Values{}
	params.Add("query", name)
	params.Add("limit", strconv.Itoa(limit))
	params.Add("countryCode", "US")

	req, err := http.NewRequestWithContext(ctx, "GET", apiBaseURL+"/search", nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = params.Encode()
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.tidal.v1+json")
	req.Header.Set("Content-Type", "application/vnd.tidal.v1+json")

	var result struct {
		Artists []ArtistResource `json:"artists"`
	}
	err = c.makeRequest(req, &result)
	if err != nil {
		return nil, err
	}

	if len(result.Artists) == 0 {
		return nil, ErrNotFound
	}
	return result.Artists, nil
}

func (c *client) getArtist(ctx context.Context, artistID string) (*ArtistResource, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	params := url.Values{}
	params.Add("countryCode", "US")

	req, err := http.NewRequestWithContext(ctx, "GET", apiBaseURL+"/artists/"+artistID, nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = params.Encode()
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.tidal.v1+json")
	req.Header.Set("Content-Type", "application/vnd.tidal.v1+json")

	var result struct {
		Resource ArtistResource `json:"resource"`
	}
	err = c.makeRequest(req, &result)
	if err != nil {
		return nil, err
	}

	return &result.Resource, nil
}

func (c *client) getArtistTopTracks(ctx context.Context, artistID string, limit int) ([]TrackResource, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	params := url.Values{}
	params.Add("countryCode", "US")
	params.Add("limit", strconv.Itoa(limit))

	req, err := http.NewRequestWithContext(ctx, "GET", apiBaseURL+"/artists/"+artistID+"/tracks", nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = params.Encode()
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.tidal.v1+json")
	req.Header.Set("Content-Type", "application/vnd.tidal.v1+json")

	var result struct {
		Data []TrackResource `json:"data"`
	}
	err = c.makeRequest(req, &result)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *client) getSimilarArtists(ctx context.Context, artistID string, limit int) ([]ArtistResource, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	params := url.Values{}
	params.Add("countryCode", "US")
	params.Add("limit", strconv.Itoa(limit))

	req, err := http.NewRequestWithContext(ctx, "GET", apiBaseURL+"/artists/"+artistID+"/similar", nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = params.Encode()
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.tidal.v1+json")
	req.Header.Set("Content-Type", "application/vnd.tidal.v1+json")

	var result struct {
		Data []ArtistResource `json:"data"`
	}
	err = c.makeRequest(req, &result)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *client) searchAlbums(ctx context.Context, albumName, artistName string, limit int) ([]AlbumResource, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	query := albumName
	if artistName != "" {
		query = artistName + " " + albumName
	}

	params := url.Values{}
	params.Add("query", query)
	params.Add("limit", strconv.Itoa(limit))
	params.Add("countryCode", "US")
	params.Add("type", "ALBUMS")

	req, err := http.NewRequestWithContext(ctx, "GET", apiBaseURL+"/search", nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = params.Encode()
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.tidal.v1+json")
	req.Header.Set("Content-Type", "application/vnd.tidal.v1+json")

	var result struct {
		Albums []AlbumResource `json:"albums"`
	}
	err = c.makeRequest(req, &result)
	if err != nil {
		return nil, err
	}

	if len(result.Albums) == 0 {
		return nil, ErrNotFound
	}
	return result.Albums, nil
}

func (c *client) searchTracks(ctx context.Context, trackName, artistName string, limit int) ([]TrackResource, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	query := trackName
	if artistName != "" {
		query = artistName + " " + trackName
	}

	params := url.Values{}
	params.Add("query", query)
	params.Add("limit", strconv.Itoa(limit))
	params.Add("countryCode", "US")
	params.Add("type", "TRACKS")

	req, err := http.NewRequestWithContext(ctx, "GET", apiBaseURL+"/search", nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = params.Encode()
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.tidal.v1+json")
	req.Header.Set("Content-Type", "application/vnd.tidal.v1+json")

	var result struct {
		Tracks []TrackResource `json:"tracks"`
	}
	err = c.makeRequest(req, &result)
	if err != nil {
		return nil, err
	}

	if len(result.Tracks) == 0 {
		return nil, ErrNotFound
	}
	return result.Tracks, nil
}

func (c *client) getArtistBio(ctx context.Context, artistID string) (string, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	params := url.Values{}
	params.Add("countryCode", "US")

	req, err := http.NewRequestWithContext(ctx, "GET", apiBaseURL+"/artists/"+artistID+"/bio", nil)
	if err != nil {
		return "", err
	}
	req.URL.RawQuery = params.Encode()
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.tidal.v1+json")
	req.Header.Set("Content-Type", "application/vnd.tidal.v1+json")

	var result struct {
		Text string `json:"text"`
	}
	err = c.makeRequest(req, &result)
	if err != nil {
		return "", err
	}

	if result.Text == "" {
		return "", ErrNotFound
	}
	return result.Text, nil
}

func (c *client) getAlbumReview(ctx context.Context, albumID string) (string, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	params := url.Values{}
	params.Add("countryCode", "US")

	req, err := http.NewRequestWithContext(ctx, "GET", apiBaseURL+"/albums/"+albumID+"/review", nil)
	if err != nil {
		return "", err
	}
	req.URL.RawQuery = params.Encode()
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.tidal.v1+json")
	req.Header.Set("Content-Type", "application/vnd.tidal.v1+json")

	var result struct {
		Text string `json:"text"`
	}
	err = c.makeRequest(req, &result)
	if err != nil {
		return "", err
	}

	if result.Text == "" {
		return "", ErrNotFound
	}
	return result.Text, nil
}

func (c *client) getTrackRadio(ctx context.Context, trackID string, limit int) ([]TrackResource, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	params := url.Values{}
	params.Add("countryCode", "US")
	params.Add("limit", strconv.Itoa(limit))

	req, err := http.NewRequestWithContext(ctx, "GET", apiBaseURL+"/tracks/"+trackID+"/radio", nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = params.Encode()
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.tidal.v1+json")
	req.Header.Set("Content-Type", "application/vnd.tidal.v1+json")

	var result struct {
		Data []TrackResource `json:"data"`
	}
	err = c.makeRequest(req, &result)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *client) getToken(ctx context.Context) (string, error) {
	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()

	// Return cached token if still valid (with 1 minute buffer)
	if c.token != "" && time.Now().Add(time.Minute).Before(c.tokenExp) {
		return c.token, nil
	}

	// Request new token
	payload := url.Values{}
	payload.Add("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, "POST", authTokenURL, strings.NewReader(payload.Encode()))
	if err != nil {
		return "", err
	}

	auth := c.id + ":" + c.secret
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	log.Trace(ctx, "Requesting Tidal OAuth token")
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tidal: failed to get token: %s", string(data))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(data, &tokenResp); err != nil {
		return "", fmt.Errorf("tidal: failed to parse token response: %w", err)
	}

	c.token = tokenResp.AccessToken
	c.tokenExp = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	log.Trace(ctx, "Obtained Tidal OAuth token", "expiresIn", tokenResp.ExpiresIn)

	return c.token, nil
}

func (c *client) makeRequest(req *http.Request, response any) error {
	log.Trace(req.Context(), fmt.Sprintf("Sending Tidal %s request", req.Method), "url", req.URL)
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusMultiStatus {
		return c.parseError(data, resp.StatusCode)
	}

	return json.Unmarshal(data, response)
}

func (c *client) parseError(data []byte, statusCode int) error {
	var errResp ErrorResponse
	if err := json.Unmarshal(data, &errResp); err != nil {
		return fmt.Errorf("tidal error (status %d): %s", statusCode, string(data))
	}
	if len(errResp.Errors) > 0 {
		return fmt.Errorf("tidal error (%s): %s", errResp.Errors[0].Code, errResp.Errors[0].Detail)
	}
	return fmt.Errorf("tidal error (status %d)", statusCode)
}

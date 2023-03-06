package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type webhookError struct {
	Code    int
	Message string
}

func (e *webhookError) Error() string {
	return fmt.Sprintf("webhook error(%d): %s", e.Code, e.Message)
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

func newClient(url, apiKey string, hc httpDoer) *client {
	return &client{apiKey, url, hc}
}

type client struct {
	apiKey string
	url    string
	hc     httpDoer
}

type ScrobbleInfo struct {
	artist      string
	track       string
	album       string
	trackNumber int
	mbid        string
	duration    int
	albumArtist string
	timestamp   time.Time
}

type webhookResponse struct {
	Error    int    `json:"error"`
	Message  string `json:"message"`
	UserName string `json:"userName"`
}

func (c *client) validateToken(ctx context.Context, token string) (*webhookResponse, error) {
	params := map[string]interface{}{
		"token": token,
	}

	return c.makeRequest(ctx, "/validate", params)
}

func (c *client) scrobble(ctx context.Context, sessionKey string, isSubmission bool, info ScrobbleInfo) error {
	params := map[string]interface{}{
		"artist":      info.artist,
		"track":       info.track,
		"album":       info.album,
		"trackNumber": info.trackNumber,
		"mbid":        info.mbid,
		"duration":    info.duration,
		"albumArtist": info.albumArtist,
		"token":       sessionKey,
	}

	if isSubmission {
		params["timestamp"] = info.timestamp.Unix()
	}

	_, err := c.makeRequest(ctx, "/scrobble", params)
	return err
}

func (c *client) makeRequest(ctx context.Context, endpoint string, params map[string]interface{}) (*webhookResponse, error) {
	params["apiKey"] = c.apiKey

	b, _ := json.Marshal(params)

	req, err := http.NewRequestWithContext(ctx, "POST", c.url+endpoint, bytes.NewBuffer(b))
	req.Header.Add("Content-Type", "application/json; charset=UTF-8")
	if err != nil {
		return nil, err
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var response webhookResponse
	decoder := json.NewDecoder(resp.Body)
	jsonErr := decoder.Decode(&response)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("webhook http status: (%d)", resp.StatusCode)
	} else if jsonErr != nil {
		return nil, jsonErr
	} else if response.Error != 0 {
		return &response, &webhookError{Code: response.Error, Message: response.Message}
	}

	return &response, nil
}

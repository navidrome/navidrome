package deezer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

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

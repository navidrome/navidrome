package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/log"
)

const apiBaseUrl = "https://api.spotify.com/v1/"

var (
	ErrNotFound = errors.New("spotify: not found")
)

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewClient(id, secret string, hc httpDoer) *Client {
	return &Client{id, secret, hc}
}

type Client struct {
	id     string
	secret string
	hc     httpDoer
}

func (c *Client) SearchArtists(ctx context.Context, name string, limit int) ([]Artist, error) {
	token, err := c.authorize(ctx)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Add("type", "artist")
	params.Add("q", name)
	params.Add("offset", "0")
	params.Add("limit", strconv.Itoa(limit))
	req, _ := http.NewRequest("GET", apiBaseUrl+"search", nil)
	req.URL.RawQuery = params.Encode()
	req.Header.Add("Authorization", "Bearer "+token)

	var results SearchResults
	err = c.makeRequest(req, &results)
	if err != nil {
		return nil, err
	}

	if len(results.Artists.Items) == 0 {
		return nil, ErrNotFound
	}
	return results.Artists.Items, err
}

func (c *Client) authorize(ctx context.Context) (string, error) {
	payload := url.Values{}
	payload.Add("grant_type", "client_credentials")

	encodePayload := payload.Encode()
	req, _ := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(encodePayload))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(encodePayload)))
	auth := c.id + ":" + c.secret
	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))

	response := map[string]interface{}{}
	err := c.makeRequest(req, &response)
	if err != nil {
		return "", err
	}

	if v, ok := response["access_token"]; ok {
		return v.(string), nil
	}
	log.Error(ctx, "Invalid spotify response", "resp", response)
	return "", errors.New("invalid response")
}

func (c *Client) makeRequest(req *http.Request, response interface{}) error {
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return c.parseError(data)
	}

	return json.Unmarshal(data, response)
}

func (c *Client) parseError(data []byte) error {
	var e Error
	err := json.Unmarshal(data, &e)
	if err != nil {
		return err
	}
	return fmt.Errorf("spotify error(%s): %s", e.Code, e.Message)
}

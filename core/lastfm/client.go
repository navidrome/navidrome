package lastfm

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	apiBaseUrl = "https://ws.audioscrobbler.com/2.0/"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewClient(apiKey string, lang string, hc HttpClient) *Client {
	return &Client{apiKey, lang, hc}
}

type Client struct {
	apiKey string
	lang   string
	hc     HttpClient
}

// TODO SimilarArtists()
func (c *Client) ArtistGetInfo(ctx context.Context, name string) (*Artist, error) {
	params := url.Values{}
	params.Add("method", "artist.getInfo")
	params.Add("format", "json")
	params.Add("api_key", c.apiKey)
	params.Add("artist", name)
	params.Add("lang", c.lang)
	req, _ := http.NewRequest("GET", apiBaseUrl, nil)
	req.URL.RawQuery = params.Encode()

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, c.parseError(data)
	}

	var response Response
	err = json.Unmarshal(data, &response)
	return &response.Artist, err
}

func (c *Client) parseError(data []byte) error {
	var e Error
	err := json.Unmarshal(data, &e)
	if err != nil {
		return err
	}
	return fmt.Errorf("last.fm error(%d): %s", e.Code, e.Message)
}

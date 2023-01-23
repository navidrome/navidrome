package radiobrowser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/navidrome/navidrome/consts"
)

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewClient(baseURL string, hc httpDoer) *Client {
	return &Client{baseURL, hc}
}

type Client struct {
	baseURL string
	hc      httpDoer
}

func (c *Client) GetAllRadios(ctx context.Context) (*RadioStations, error) {
	params := url.Values{}
	params.Add("hidebroken", "true")

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/json/stations", nil)
	req.URL.RawQuery = params.Encode()
	req.Header.Add("User-Agent", consts.AppName+"/"+consts.Version)

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	var stations RadioStations
	err = decoder.Decode(&stations)
	if err != nil {
		return nil, err
	}
	return &stations, nil
}

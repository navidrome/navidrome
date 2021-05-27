package lastfm

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

const (
	apiBaseUrl = "https://ws.audioscrobbler.com/2.0/"
)

type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("last.fm error(%d): %s", e.Code, e.Message)
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewClient(apiKey string, lang string, hc httpDoer) *Client {
	return &Client{apiKey, lang, hc}
}

type Client struct {
	apiKey string
	lang   string
	hc     httpDoer
}

func (c *Client) makeRequest(params url.Values) (*Response, error) {
	params.Add("format", "json")
	params.Add("api_key", c.apiKey)

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

	var response Response
	jsonErr := json.Unmarshal(data, &response)

	if resp.StatusCode != 200 && jsonErr != nil {
		return nil, fmt.Errorf("last.fm http status: (%d)", resp.StatusCode)
	}

	if jsonErr != nil {
		return nil, jsonErr
	}

	if response.Error != 0 {
		return &response, &Error{Code: response.Error, Message: response.Message}
	}

	return &response, nil
}

func (c *Client) ArtistGetInfo(ctx context.Context, name string, mbid string) (*Artist, error) {
	params := url.Values{}
	params.Add("method", "artist.getInfo")
	params.Add("artist", name)
	params.Add("mbid", mbid)
	params.Add("lang", c.lang)
	response, err := c.makeRequest(params)
	if err != nil {
		return nil, err
	}
	return &response.Artist, nil
}

func (c *Client) ArtistGetSimilar(ctx context.Context, name string, mbid string, limit int) (*SimilarArtists, error) {
	params := url.Values{}
	params.Add("method", "artist.getSimilar")
	params.Add("artist", name)
	params.Add("mbid", mbid)
	params.Add("limit", strconv.Itoa(limit))
	response, err := c.makeRequest(params)
	if err != nil {
		return nil, err
	}
	return &response.SimilarArtists, nil
}

func (c *Client) ArtistGetTopTracks(ctx context.Context, name string, mbid string, limit int) (*TopTracks, error) {
	params := url.Values{}
	params.Add("method", "artist.getTopTracks")
	params.Add("artist", name)
	params.Add("mbid", mbid)
	params.Add("limit", strconv.Itoa(limit))
	response, err := c.makeRequest(params)
	if err != nil {
		return nil, err
	}
	return &response.TopTracks, nil
}

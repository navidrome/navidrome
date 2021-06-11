package lastfm

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/utils"
)

const (
	apiBaseUrl = "https://ws.audioscrobbler.com/2.0/"
)

type lastFMError struct {
	Code    int
	Message string
}

func (e *lastFMError) Error() string {
	return fmt.Sprintf("last.fm error(%d): %s", e.Code, e.Message)
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewClient(apiKey string, secret string, lang string, hc httpDoer) *Client {
	return &Client{apiKey, secret, lang, hc}
}

type Client struct {
	apiKey string
	secret string
	lang   string
	hc     httpDoer
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

func (c *Client) GetToken(ctx context.Context) (string, error) {
	params := url.Values{}
	params.Add("method", "auth.getToken")
	c.sign(params)
	response, err := c.makeRequest(params)
	if err != nil {
		return "", err
	}
	return response.Token, nil
}

func (c *Client) GetSession(ctx context.Context, token string) (string, error) {
	params := url.Values{}
	params.Add("method", "auth.getSession")
	params.Add("token", token)
	c.sign(params)
	response, err := c.makeRequest(params)
	if err != nil {
		return "", err
	}
	return response.Session.Key, nil
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
	decoder := json.NewDecoder(resp.Body)

	var response Response
	jsonErr := decoder.Decode(&response)
	if resp.StatusCode != 200 && jsonErr != nil {
		return nil, fmt.Errorf("last.fm http status: (%d)", resp.StatusCode)
	}
	if jsonErr != nil {
		return nil, jsonErr
	}
	if response.Error != 0 {
		return &response, &lastFMError{Code: response.Error, Message: response.Message}
	}

	return &response, nil
}

func (c *Client) sign(params url.Values) {
	// the parameters must be in order before hashing
	keys := make([]string, 0, len(params))
	for k := range params {
		if utils.StringInSlice(k, []string{"format", "callback"}) {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	msg := strings.Builder{}
	for _, k := range keys {
		msg.WriteString(k)
		msg.WriteString(params[k][0])
	}
	msg.WriteString(c.secret)
	hash := md5.Sum([]byte(msg.String()))
	params.Add("api_sig", hex.EncodeToString(hash[:]))
}

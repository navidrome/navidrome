package lastfm

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/log"
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

func newClient(apiKey string, secret string, lang string, hc httpDoer) *client {
	return &client{apiKey, secret, lang, hc}
}

type client struct {
	apiKey string
	secret string
	lang   string
	hc     httpDoer
}

func (c *client) albumGetInfo(ctx context.Context, name string, artist string, mbid string) (*Album, error) {
	params := url.Values{}
	params.Add("method", "album.getInfo")
	params.Add("album", name)
	params.Add("artist", artist)
	params.Add("mbid", mbid)
	params.Add("lang", c.lang)
	response, err := c.makeRequest(ctx, http.MethodGet, params, false)
	if err != nil {
		return nil, err
	}
	return &response.Album, nil
}

func (c *client) artistGetInfo(ctx context.Context, name string) (*Artist, error) {
	params := url.Values{}
	params.Add("method", "artist.getInfo")
	params.Add("artist", name)
	params.Add("lang", c.lang)
	response, err := c.makeRequest(ctx, http.MethodGet, params, false)
	if err != nil {
		return nil, err
	}
	return &response.Artist, nil
}

func (c *client) artistGetSimilar(ctx context.Context, name string, limit int) (*SimilarArtists, error) {
	params := url.Values{}
	params.Add("method", "artist.getSimilar")
	params.Add("artist", name)
	params.Add("limit", strconv.Itoa(limit))
	response, err := c.makeRequest(ctx, http.MethodGet, params, false)
	if err != nil {
		return nil, err
	}
	return &response.SimilarArtists, nil
}

func (c *client) artistGetTopTracks(ctx context.Context, name string, limit int) (*TopTracks, error) {
	params := url.Values{}
	params.Add("method", "artist.getTopTracks")
	params.Add("artist", name)
	params.Add("limit", strconv.Itoa(limit))
	response, err := c.makeRequest(ctx, http.MethodGet, params, false)
	if err != nil {
		return nil, err
	}
	return &response.TopTracks, nil
}

func (c *client) GetToken(ctx context.Context) (string, error) {
	params := url.Values{}
	params.Add("method", "auth.getToken")
	c.sign(params)
	response, err := c.makeRequest(ctx, http.MethodGet, params, true)
	if err != nil {
		return "", err
	}
	return response.Token, nil
}

func (c *client) getSession(ctx context.Context, token string) (string, error) {
	params := url.Values{}
	params.Add("method", "auth.getSession")
	params.Add("token", token)
	response, err := c.makeRequest(ctx, http.MethodGet, params, true)
	if err != nil {
		return "", err
	}
	return response.Session.Key, nil
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

func (c *client) updateNowPlaying(ctx context.Context, sessionKey string, info ScrobbleInfo) error {
	params := url.Values{}
	params.Add("method", "track.updateNowPlaying")
	params.Add("artist", info.artist)
	params.Add("track", info.track)
	params.Add("album", info.album)
	params.Add("trackNumber", strconv.Itoa(info.trackNumber))
	params.Add("mbid", info.mbid)
	params.Add("duration", strconv.Itoa(info.duration))
	params.Add("albumArtist", info.albumArtist)
	params.Add("sk", sessionKey)
	resp, err := c.makeRequest(ctx, http.MethodPost, params, true)
	if err != nil {
		return err
	}
	if resp.NowPlaying.IgnoredMessage.Code != "0" {
		log.Warn(ctx, "LastFM: NowPlaying was ignored", "code", resp.NowPlaying.IgnoredMessage.Code,
			"text", resp.NowPlaying.IgnoredMessage.Text)
	}
	return nil
}

func (c *client) scrobble(ctx context.Context, sessionKey string, info ScrobbleInfo) error {
	params := url.Values{}
	params.Add("method", "track.scrobble")
	params.Add("timestamp", strconv.FormatInt(info.timestamp.Unix(), 10))
	params.Add("artist", info.artist)
	params.Add("track", info.track)
	params.Add("album", info.album)
	params.Add("trackNumber", strconv.Itoa(info.trackNumber))
	params.Add("mbid", info.mbid)
	params.Add("duration", strconv.Itoa(info.duration))
	params.Add("albumArtist", info.albumArtist)
	params.Add("sk", sessionKey)
	resp, err := c.makeRequest(ctx, http.MethodPost, params, true)
	if err != nil {
		return err
	}
	if resp.Scrobbles.Scrobble.IgnoredMessage.Code != "0" {
		log.Warn(ctx, "LastFM: scrobble was ignored", "code", resp.Scrobbles.Scrobble.IgnoredMessage.Code,
			"text", resp.Scrobbles.Scrobble.IgnoredMessage.Text, "info", info)
	}
	if resp.Scrobbles.Attr.Accepted != 1 {
		log.Warn(ctx, "LastFM: scrobble was not accepted", "code", resp.Scrobbles.Scrobble.IgnoredMessage.Code,
			"text", resp.Scrobbles.Scrobble.IgnoredMessage.Text, "info", info)
	}
	return nil
}

func (c *client) makeRequest(ctx context.Context, method string, params url.Values, signed bool) (*Response, error) {
	params.Add("format", "json")
	params.Add("api_key", c.apiKey)

	if signed {
		c.sign(params)
	}

	req, _ := http.NewRequestWithContext(ctx, method, apiBaseUrl, nil)
	req.URL.RawQuery = params.Encode()

	log.Trace(ctx, fmt.Sprintf("Sending Last.fm %s request", req.Method), "url", req.URL)
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

func (c *client) sign(params url.Values) {
	// the parameters must be in order before hashing
	keys := make([]string, 0, len(params))
	for k := range params {
		if slices.Contains([]string{"format", "callback"}, k) {
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

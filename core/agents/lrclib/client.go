package lrclib

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

const apiBaseUrl = "https://lrclib.net/api/"

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

func newClient(hc httpDoer) *client {
	return &client{hc}
}

type client struct {
	hc httpDoer
}

type lyricInfo struct {
	Id           int    `json:"id"`
	Instrumental bool   `json:"instrumental"`
	PlainLyrics  string `json:"plainLyrics,omitempty"`
	SyncedLyrics string `json:"syncedLyrics,omitempty"`
}

type lrclibError struct {
	Code    int    `json:"code"`
	Name    string `json:"name,omitempty"`
	Message string `json:"message,omitempty"`
}

func (e lrclibError) Error() string {
	return fmt.Sprintf("lrclib error(%d, %s): %s", e.Code, e.Name, e.Message)
}

func (c *client) getLyrics(ctx context.Context, trackName, artistName, albumName string, durationSec float32) (*lyricInfo, error) {
	params := url.Values{}
	params.Add("track_name", trackName)
	params.Add("artist_name", artistName)
	params.Add("album_name", albumName)
	params.Add("duration", strconv.Itoa(int(durationSec)))

	req, _ := http.NewRequestWithContext(ctx, "GET", apiBaseUrl+"get", nil)
	req.URL.RawQuery = params.Encode()

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, c.parseError(data)
	}

	var lyricData lyricInfo
	err = json.Unmarshal(data, &lyricData)
	if err != nil {
		return nil, err
	}

	return &lyricData, nil
}

func (c *client) parseError(data []byte) error {
	var e lrclibError
	err := json.Unmarshal(data, &e)
	if err != nil {
		return err
	}
	return e
}

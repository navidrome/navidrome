package listenbrainz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/navidrome/navidrome/log"
)

const (
	apiBaseUrl = "https://api.listenbrainz.org/1/"
)

type listenBrainzError struct {
	Code    int
	Message string
}

func (e *listenBrainzError) Error() string {
	return fmt.Sprintf("ListenBrainz error(%d): %s", e.Code, e.Message)
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewClient(hc httpDoer) *Client {
	return &Client{hc}
}

type Client struct {
	hc httpDoer
}

type listenBrainzResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error"`
	Status  string `json:"status"`
	Valid   bool   `json:"valid"`
	User    string `json:"user_name"`
}

type listenBrainzRequest struct {
	ApiKey string
	Body   listenBrainzRequestBody
}

type listenBrainzRequestBody struct {
	ListenType listenType   `json:"listen_type,omitempty"`
	ListenInfo []listenInfo `json:"payload,omitempty"`
}

type listenType string

const (
	Scrobble   listenType = "single"
	NowPlaying listenType = "playing_now"
)

type listenInfo struct {
	Timestamp int           `json:"listened_at,omitempty"`
	Track     trackMetadata `json:"track_metadata,omitempty"`
}

type trackMetadata struct {
	Artist         string             `json:"artist_name,omitempty"`
	Track          string             `json:"track_name,omitempty"`
	Album          string             `json:"release_name,omitempty"`
	AdditionalInfo additionalMetadata `json:"additional_info,omitempty"`
}

type additionalMetadata struct {
	TrackNumber  int      `json:"tracknumber,omitempty"`
	MbzTrackID   string   `json:"track_mbid,omitempty"`
	MbzArtistIDs []string `json:"artist_mbids,omitempty"`
	MbzAlbumID   string   `json:"release_mbid,omitempty"`
	Player       string   `json:"listening_from,omitempty"`
}

func (c *Client) ValidateToken(ctx context.Context, apiKey string) (*listenBrainzResponse, error) {
	r := &listenBrainzRequest{
		ApiKey: apiKey,
	}
	response, err := c.makeRequest(http.MethodGet, "validate-token", r)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) UpdateNowPlaying(ctx context.Context, apiKey string, li listenInfo) error {
	r := &listenBrainzRequest{
		ApiKey: apiKey,
		Body: listenBrainzRequestBody{
			ListenType: NowPlaying,
			ListenInfo: []listenInfo{li},
		},
	}

	resp, err := c.makeRequest(http.MethodPost, "submit-listens", r)
	if err != nil {
		return err
	}
	if resp.Status != "ok" {
		log.Warn(ctx, "ListenBrainz: NowPlaying was not accepted", "status", resp.Status)
	}
	return nil
}

func (c *Client) Scrobble(ctx context.Context, apiKey string, li listenInfo) error {
	r := &listenBrainzRequest{
		ApiKey: apiKey,
		Body: listenBrainzRequestBody{
			ListenType: Scrobble,
			ListenInfo: []listenInfo{li},
		},
	}
	resp, err := c.makeRequest(http.MethodPost, "submit-listens", r)
	if err != nil {
		return err
	}
	if resp.Status != "ok" {
		log.Warn(ctx, "ListenBrainz: Scrobble was not accepted", "status", resp.Status)
	}
	return nil
}

func (c *Client) makeRequest(method string, endpoint string, r *listenBrainzRequest) (*listenBrainzResponse, error) {
	b, _ := json.Marshal(r.Body)
	req, _ := http.NewRequest(method, apiBaseUrl+endpoint, bytes.NewBuffer(b))

	if r.ApiKey != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Token %s", r.ApiKey))
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	var response listenBrainzResponse
	jsonErr := decoder.Decode(&response)
	if resp.StatusCode != 200 && jsonErr != nil {
		return nil, fmt.Errorf("ListenBrainz http status: (%d)", resp.StatusCode)
	}
	if jsonErr != nil {
		return nil, jsonErr
	}
	if response.Code != 0 && response.Code != 200 {
		return &response, &listenBrainzError{Code: response.Code, Message: response.Error}
	}

	return &response, nil
}

package listenbrainz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/navidrome/navidrome/core/external_playlists"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
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

func newClient(baseURL string, hc httpDoer) *client {
	return &client{baseURL, hc}
}

type client struct {
	baseURL string
	hc      httpDoer
}

type listenBrainzResponse struct {
	Code          int               `json:"code"`
	Message       string            `json:"message"`
	Error         string            `json:"error"`
	Status        string            `json:"status"`
	Valid         bool              `json:"valid"`
	UserName      string            `json:"user_name"`
	PlaylistCount int               `json:"playlist_count"`
	Playlists     []overallPlaylist `json:"playlists,omitempty"`
	Playlist      lbPlaylist        `json:"playlist"`
}

type listenBrainzRequest struct {
	ApiKey string
	Body   *listenBrainzRequestBody
}

type overallPlaylist struct {
	Playlist lbPlaylist `json:"playlist"`
}

type lbPlaylist struct {
	Annotation string       `json:"annotation"`
	Creator    string       `json:"creator"`
	Date       time.Time    `json:"date"`
	Identifier string       `json:"identifier"`
	Title      string       `json:"title"`
	Extension  plsExtension `json:"extension"`
	Tracks     []lbTrack    `json:"track"`
}

type plsExtension struct {
	Extension playlistExtension `json:"https://musicbrainz.org/doc/jspf#playlist"`
}

type playlistExtension struct {
	Collaborators []string  `json:"collaborators"`
	CreatedFor    string    `json:"created_for"`
	LastModified  time.Time `json:"last_modified_at"`
	Public        bool      `json:"public"`
}

type lbTrack struct {
	Creator    string `json:"creator"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
}

type listenBrainzRequestBody struct {
	ListenType listenType   `json:"listen_type,omitempty"`
	Payload    []listenInfo `json:"payload,omitempty"`
}

type listenType string

const (
	Single     listenType = "single"
	PlayingNow listenType = "playing_now"
)

type listenInfo struct {
	ListenedAt    int           `json:"listened_at,omitempty"`
	TrackMetadata trackMetadata `json:"track_metadata,omitempty"`
}

type trackMetadata struct {
	ArtistName     string         `json:"artist_name,omitempty"`
	TrackName      string         `json:"track_name,omitempty"`
	ReleaseName    string         `json:"release_name,omitempty"`
	AdditionalInfo additionalInfo `json:"additional_info,omitempty"`
}

type additionalInfo struct {
	SubmissionClient        string   `json:"submission_client,omitempty"`
	SubmissionClientVersion string   `json:"submission_client_version,omitempty"`
	TrackNumber             int      `json:"tracknumber,omitempty"`
	TrackMbzID              string   `json:"track_mbid,omitempty"`
	ArtistMbzIDs            []string `json:"artist_mbids,omitempty"`
	ReleaseMbID             string   `json:"release_mbid,omitempty"`
}

func (c *client) validateToken(ctx context.Context, apiKey string) (*listenBrainzResponse, error) {
	r := &listenBrainzRequest{
		ApiKey: apiKey,
	}
	response, err := c.makeRequest(ctx, http.MethodGet, "validate-token", "", r)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *client) updateNowPlaying(ctx context.Context, apiKey string, li listenInfo) error {
	r := &listenBrainzRequest{
		ApiKey: apiKey,
		Body: &listenBrainzRequestBody{
			ListenType: PlayingNow,
			Payload:    []listenInfo{li},
		},
	}

	resp, err := c.makeRequest(ctx, http.MethodPost, "submit-listens", "", r)
	if err != nil {
		return err
	}
	if resp.Status != "ok" {
		log.Warn(ctx, "ListenBrainz: NowPlaying was not accepted", "status", resp.Status)
	}
	return nil
}

func (c *client) scrobble(ctx context.Context, apiKey string, li listenInfo) error {
	r := &listenBrainzRequest{
		ApiKey: apiKey,
		Body: &listenBrainzRequestBody{
			ListenType: Single,
			Payload:    []listenInfo{li},
		},
	}
	resp, err := c.makeRequest(ctx, http.MethodPost, "submit-listens", "", r)
	if err != nil {
		return err
	}
	if resp.Status != "ok" {
		log.Warn(ctx, "ListenBrainz: Scrobble was not accepted", "status", resp.Status)
	}
	return nil
}

func (c *client) getPlaylists(ctx context.Context, offset, count int, apiKey, user, plsType string) (*listenBrainzResponse, error) {
	r := &listenBrainzRequest{
		ApiKey: apiKey,
		Body:   nil,
	}

	var endpoint string

	switch plsType {
	case "user":
		endpoint = "user/" + user + "/playlists"
	case "created":
		endpoint = "user/" + user + "/playlists/createdfor"
	case "collab":
		endpoint = "user/" + user + "/playlists/collaborator"
	default:
		return nil, external_playlists.ErrorUnsupportedType
	}

	extra := fmt.Sprintf("?count=%d&offset=%d", count, offset)

	resp, err := c.makeRequest(ctx, http.MethodGet, endpoint, extra, r)

	if err != nil {
		return nil, err
	}

	return resp, err
}

func (c *client) getPlaylist(ctx context.Context, apiKey, plsId string) (*listenBrainzResponse, error) {
	r := &listenBrainzRequest{
		ApiKey: apiKey,
	}

	endpoint := fmt.Sprintf("playlist/%s", plsId)

	resp, err := c.makeRequest(ctx, http.MethodGet, endpoint, "", r)

	if resp.Code == 404 {
		return nil, model.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *client) path(endpoint string) (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, endpoint)
	return u.String(), nil
}

func (c *client) makeRequest(ctx context.Context, method string, endpoint string, query string, r *listenBrainzRequest) (*listenBrainzResponse, error) {
	uri, err := c.path(endpoint)
	if err != nil {
		return nil, err
	}

	if query != "" {
		uri += query
	}

	var req *http.Request

	if r.Body != nil {
		b, _ := json.Marshal(r.Body)
		req, _ = http.NewRequestWithContext(ctx, method, uri, bytes.NewBuffer(b))
	} else {
		req, _ = http.NewRequestWithContext(ctx, method, uri, nil)
	}

	req.Header.Add("Content-Type", "application/json; charset=UTF-8")

	if r.ApiKey != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Token %s", r.ApiKey))
	}

	log.Trace(ctx, fmt.Sprintf("Sending ListenBrainz %s request", req.Method), "url", req.URL)
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	var response listenBrainzResponse
	jsonErr := decoder.Decode(&response)
	if resp.StatusCode != 200 && jsonErr != nil {
		return nil, fmt.Errorf("ListenBrainz: HTTP Error, Status: (%d)", resp.StatusCode)
	}
	if jsonErr != nil {
		return nil, jsonErr
	}
	if response.Code != 0 && response.Code != 200 {
		return &response, &listenBrainzError{Code: response.Code, Message: response.Error}
	}

	return &response, nil
}

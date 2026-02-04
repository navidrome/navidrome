package listenbrainz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/navidrome/navidrome/log"
)

const (
	lbzApiUrl = "https://api.listenbrainz.org/1/"
	labsBase  = "https://labs.api.listenbrainz.org/"
	// There are a couple of algorithms from https://labs.api.listenbrainz.org/similar-artists
	algorithm = "session_based_days_9000_session_300_contribution_5_threshold_15_limit_50_skip_30"
)

var (
	ErrorNotFound = errors.New("listenbrainz: not found")
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
	Code     int    `json:"code"`
	Message  string `json:"message"`
	Error    string `json:"error"`
	Status   string `json:"status"`
	Valid    bool   `json:"valid"`
	UserName string `json:"user_name"`
}

type listenBrainzRequest struct {
	ApiKey string
	Body   listenBrainzRequestBody
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
	ArtistNames             []string `json:"artist_names,omitempty"`
	ArtistMBIDs             []string `json:"artist_mbids,omitempty"`
	RecordingMBID           string   `json:"recording_mbid,omitempty"`
	ReleaseMBID             string   `json:"release_mbid,omitempty"`
	ReleaseGroupMBID        string   `json:"release_group_mbid,omitempty"`
	DurationMs              int      `json:"duration_ms,omitempty"`
}

func (c *client) validateToken(ctx context.Context, apiKey string) (*listenBrainzResponse, error) {
	r := &listenBrainzRequest{
		ApiKey: apiKey,
	}
	response, err := c.makeAuthenticatedRequest(ctx, http.MethodGet, "validate-token", r)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *client) updateNowPlaying(ctx context.Context, apiKey string, li listenInfo) error {
	r := &listenBrainzRequest{
		ApiKey: apiKey,
		Body: listenBrainzRequestBody{
			ListenType: PlayingNow,
			Payload:    []listenInfo{li},
		},
	}

	resp, err := c.makeAuthenticatedRequest(ctx, http.MethodPost, "submit-listens", r)
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
		Body: listenBrainzRequestBody{
			ListenType: Single,
			Payload:    []listenInfo{li},
		},
	}
	resp, err := c.makeAuthenticatedRequest(ctx, http.MethodPost, "submit-listens", r)
	if err != nil {
		return err
	}
	if resp.Status != "ok" {
		log.Warn(ctx, "ListenBrainz: Scrobble was not accepted", "status", resp.Status)
	}
	return nil
}

func (c *client) path(endpoint string) (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, endpoint)
	return u.String(), nil
}

func (c *client) makeAuthenticatedRequest(ctx context.Context, method string, endpoint string, r *listenBrainzRequest) (*listenBrainzResponse, error) {
	b, _ := json.Marshal(r.Body)
	uri, err := c.path(endpoint)
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequestWithContext(ctx, method, uri, bytes.NewBuffer(b))
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

type lbzHttpError struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
}

func (c *client) makeGenericRequest(ctx context.Context, method string, endpoint string, params url.Values) (*http.Response, error) {
	req, _ := http.NewRequestWithContext(ctx, method, lbzApiUrl+endpoint, nil)
	req.Header.Add("Content-Type", "application/json; charset=UTF-8")
	req.URL.RawQuery = params.Encode()

	log.Trace(ctx, fmt.Sprintf("Sending ListenBrainz %s request", req.Method), "url", req.URL)
	resp, err := c.hc.Do(req)

	if err != nil {
		return nil, err
	}

	// On a 200 code, there is no code. Decode using using error message if it exists
	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)

		var lbzError lbzHttpError
		jsonErr := decoder.Decode(&lbzError)

		if jsonErr != nil {
			return nil, fmt.Errorf("ListenBrainz: HTTP Error, Status: (%d)", resp.StatusCode)
		}

		return nil, &listenBrainzError{Code: lbzError.Code, Message: lbzError.Error}
	}

	return resp, err
}

type artistMetadataResult struct {
	Rels struct {
		OfficialHomepage string `json:"official homepage,omitempty"`
	} `json:"rels,omitzero"`
}

func (c *client) getArtistUrl(ctx context.Context, mbid string) (string, error) {
	params := url.Values{}
	params.Add("artist_mbids", mbid)
	resp, err := c.makeGenericRequest(ctx, http.MethodGet, "metadata/artist", params)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	var response []artistMetadataResult
	jsonErr := decoder.Decode(&response)
	if jsonErr != nil {
		return "", fmt.Errorf("ListenBrainz: HTTP Error, Status: (%d)", resp.StatusCode)
	}

	if len(response) == 0 || response[0].Rels.OfficialHomepage == "" {
		return "", ErrorNotFound
	}

	return response[0].Rels.OfficialHomepage, nil
}

type trackInfo struct {
	ArtistName    string   `json:"artist_name"`
	ArtistMBIDs   []string `json:"artist_mbids"`
	DurationMs    uint32   `json:"length"`
	RecordingName string   `json:"recording_name"`
	RecordingMbid string   `json:"recording_mbid"`
	ReleaseName   string   `json:"release_name"`
	ReleaseMBID   string   `json:"release_mbid"`
}

func (c *client) getArtistTopSongs(ctx context.Context, mbid string, count int) ([]trackInfo, error) {
	resp, err := c.makeGenericRequest(ctx, http.MethodGet, "popularity/top-recordings-for-artist/"+mbid, url.Values{})
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	var response []trackInfo
	jsonErr := decoder.Decode(&response)
	if jsonErr != nil {
		return nil, fmt.Errorf("ListenBrainz: HTTP Error, Status: (%d)", resp.StatusCode)
	}

	if len(response) > count {
		return response[0:count], nil
	}

	return response, nil
}

type artist struct {
	MBID string `json:"artist_mbid"`
	Name string `json:"name"`
}

func (c *client) getSimilarArtists(ctx context.Context, mbid string) ([]artist, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, labsBase+"similar-artists/json", nil)
	req.Header.Add("Content-Type", "application/json; charset=UTF-8")
	req.URL.RawQuery = url.Values{
		"artist_mbids": []string{mbid}, "algorithm": []string{algorithm},
	}.Encode()

	log.Trace(ctx, fmt.Sprintf("Sending ListenBrainz Labs %s request", req.Method), "url", req.URL)
	resp, err := c.hc.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	var artists []artist
	jsonErr := decoder.Decode(&artists)
	if jsonErr != nil {
		return nil, fmt.Errorf("ListenBrainz: HTTP Error, Status: (%d)", resp.StatusCode)
	}

	return artists, nil
}

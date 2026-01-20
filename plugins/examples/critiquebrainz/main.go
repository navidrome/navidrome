//go:build wasip1

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/metadata"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
)

const (
	critiquebrainzBaseURL = "https://critiquebrainz.org/ws/1"
	musicbrainzBaseURL    = "https://musicbrainz.org/ws/2"
	userAgent             = "Navidrome/1.0 (CritiqueBrainz Plugin)"
	cacheTTL              = 24 * 60 * 60
	releaseGroupCacheTTL  = 7 * 24 * 60 * 60
)

var errNotFound = errors.New("not found")

type critiquebrainzPlugin struct{}

func init() {
	metadata.Register(&critiquebrainzPlugin{})
}

var _ metadata.AlbumInfoProvider = (*critiquebrainzPlugin)(nil)

type reviewResponse struct {
	AverageRating *averageRating `json:"average_rating,omitempty"`
	Reviews       []review       `json:"reviews"`
}

type averageRating struct {
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}

type review struct {
	Text   string `json:"text"`
	Rating *int   `json:"rating"`
	User   user   `json:"user"`
}

type user struct {
	DisplayName string `json:"display_name"`
}

type musicBrainzRelease struct {
	ReleaseGroup struct {
		ID string `json:"id"`
	} `json:"release-group"`
}

func getReleaseGroupID(releaseID string) (string, error) {
	cacheKey := "release_to_rg_" + releaseID
	if cached, exists, err := host.CacheGetString(cacheKey); err == nil && exists {
		if cached == "" {
			return "", errNotFound
		}
		return cached, nil
	}

	req := pdk.NewHTTPRequest(pdk.MethodGet, fmt.Sprintf("%s/release/%s?inc=release-groups&fmt=json", musicbrainzBaseURL, releaseID))
	req.SetHeader("Accept", "application/json")
	req.SetHeader("User-Agent", userAgent)

	resp := req.Send()
	if resp.Status() != 200 {
		return "", fmt.Errorf("MusicBrainz HTTP %d", resp.Status())
	}

	var release musicBrainzRelease
	if err := json.Unmarshal(resp.Body(), &release); err != nil {
		return "", err
	}

	if release.ReleaseGroup.ID == "" {
		_ = host.CacheSetString(cacheKey, "", int64(releaseGroupCacheTTL))
		return "", errNotFound
	}

	_ = host.CacheSetString(cacheKey, release.ReleaseGroup.ID, int64(releaseGroupCacheTTL))
	return release.ReleaseGroup.ID, nil
}

func (critiquebrainzPlugin) GetAlbumInfo(input metadata.AlbumRequest) (*metadata.AlbumInfoResponse, error) {
	if input.MBID == "" {
		return nil, errNotFound
	}

	releaseGroupID, err := getReleaseGroupID(input.MBID)
	if err != nil {
		if err != errNotFound {
			pdk.Log(pdk.LogWarn, fmt.Sprintf("Failed to get release group ID for MBID %s: %v", input.MBID, err))
		}
		return nil, errNotFound
	}

	cacheKey := "album_review_" + releaseGroupID
	if cached, exists, err := host.CacheGetString(cacheKey); err == nil && exists {
		if cached == "" {
			return nil, errNotFound
		}
		return &metadata.AlbumInfoResponse{
			MBID:        input.MBID,
			Name:        input.Name,
			Description: cached,
			URL:         "https://critiquebrainz.org/release-group/" + releaseGroupID,
		}, nil
	}

	description, err := fetchReviews(releaseGroupID)
	if err != nil {
		if err == errNotFound {
			_ = host.CacheSetString(cacheKey, "", int64(cacheTTL))
		}
		return nil, err
	}

	_ = host.CacheSetString(cacheKey, description, int64(cacheTTL))

	return &metadata.AlbumInfoResponse{
		MBID:        input.MBID,
		Name:        input.Name,
		Description: description,
		URL:         "https://critiquebrainz.org/release-group/" + releaseGroupID,
	}, nil
}

func fetchReviews(releaseGroupID string) (string, error) {
	params := url.Values{}
	params.Set("entity_id", releaseGroupID)
	params.Set("entity_type", "release_group")
	params.Set("limit", "5")
	params.Set("sort", "popularity")
	params.Set("sort_order", "desc")

	req := pdk.NewHTTPRequest(pdk.MethodGet, critiquebrainzBaseURL+"/review/?"+params.Encode())
	req.SetHeader("Accept", "application/json")
	req.SetHeader("User-Agent", userAgent)

	resp := req.Send()
	if resp.Status() != 200 {
		pdk.Log(pdk.LogWarn, fmt.Sprintf("HTTP error: status %d", resp.Status()))
		return "", fmt.Errorf("HTTP %d", resp.Status())
	}

	var reviewResp reviewResponse
	if err := json.Unmarshal(resp.Body(), &reviewResp); err != nil {
		return "", err
	}
	if len(reviewResp.Reviews) == 0 {
		return "", errNotFound
	}

	return formatReviews(&reviewResp), nil
}

func formatReviews(resp *reviewResponse) string {
	var sb strings.Builder

	if resp.AverageRating != nil && resp.AverageRating.Count > 0 {
		fmt.Fprintf(&sb, "Average Rating: %.1f/5 (%d ratings)\n\n", resp.AverageRating.Average, resp.AverageRating.Count)
	}

	var reviewStrings []string
	for _, r := range resp.Reviews {
		if r.Text == "" {
			continue
		}
		var reviewSb strings.Builder
		if r.Rating != nil {
			fmt.Fprintf(&reviewSb, "Rating: %d/5 ", *r.Rating)
		}
		fmt.Fprintf(&reviewSb, "by %s\n\n%s", r.User.DisplayName, r.Text)
		reviewStrings = append(reviewStrings, reviewSb.String())
	}
	sb.WriteString(strings.Join(reviewStrings, "\n---\n\n"))

	return sb.String()
}

func main() {}

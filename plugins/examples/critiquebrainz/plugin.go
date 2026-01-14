//go:build wasip1

// CritiqueBrainz plugin fetches album reviews from CritiqueBrainz.
//
// NOTE: CritiqueBrainz indexes reviews by release group MBID, not release MBID.
// Currently, the core passes req.Mbid as a release MBID, so this plugin must make
// an additional API call to MusicBrainz to resolve the release group.
// If the core were to pass a release group MBID directly (e.g., via a ReleaseGroupMbid
// field on AlbumInfoRequest), this lookup could be eliminated, reducing latency and
// API rate limit pressure on MusicBrainz.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/cache"
	"github.com/navidrome/navidrome/plugins/host/http"
)

const (
	critiquebrainzBaseURL = "https://critiquebrainz.org/ws/1"
	musicbrainzBaseURL    = "https://musicbrainz.org/ws/2"
	requestTimeoutMs      = 5000
	cacheTTL              = 24 * time.Hour
	releaseGroupCacheTTL  = 7 * 24 * time.Hour
	userAgent             = "Navidrome/1.0 (CritiqueBrainz Plugin)"
)

var (
	client       = http.NewHttpService()
	cacheService = cache.NewCacheService()
)

type CritiqueBrainzAgent struct{}

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

func getReleaseGroupID(ctx context.Context, releaseID string) (string, error) {
	cacheKey := "release_to_rg_" + releaseID
	if cached, err := cacheService.GetString(ctx, &cache.GetRequest{Key: cacheKey}); err == nil && cached.Exists {
		if cached.Value == "" {
			return "", api.ErrNotFound
		}
		return cached.Value, nil
	} else if err != nil {
		log.Printf("Error reading release group ID from cache: %v", err)
	}

	resp, err := client.Get(ctx, &http.HttpRequest{
		Url:       fmt.Sprintf("%s/release/%s?inc=release-groups&fmt=json", musicbrainzBaseURL, releaseID),
		Headers:   map[string]string{"Accept": "application/json", "User-Agent": userAgent},
		TimeoutMs: requestTimeoutMs,
	})
	if err != nil {
		return "", err
	}
	if resp.Status != 200 {
		return "", fmt.Errorf("MusicBrainz HTTP %d", resp.Status)
	}

	var release musicBrainzRelease
	if err := json.Unmarshal(resp.Body, &release); err != nil {
		return "", err
	}

	if release.ReleaseGroup.ID == "" {
		if _, cacheErr := cacheService.SetString(ctx, &cache.SetStringRequest{
			Key:        cacheKey,
			Value:      "",
			TtlSeconds: int64(releaseGroupCacheTTL.Seconds()),
		}); cacheErr != nil {
			log.Printf("Failed to cache empty release group ID: %v", cacheErr)
		}
		return "", api.ErrNotFound
	}

	if _, cacheErr := cacheService.SetString(ctx, &cache.SetStringRequest{
		Key:        cacheKey,
		Value:      release.ReleaseGroup.ID,
		TtlSeconds: int64(releaseGroupCacheTTL.Seconds()),
	}); cacheErr != nil {
		log.Printf("Failed to cache release group ID: %v", cacheErr)
	}
	return release.ReleaseGroup.ID, nil
}

func (CritiqueBrainzAgent) GetAlbumInfo(ctx context.Context, req *api.AlbumInfoRequest) (*api.AlbumInfoResponse, error) {
	if req.Mbid == "" {
		return nil, api.ErrNotFound
	}

	releaseGroupID, err := getReleaseGroupID(ctx, req.Mbid)
	if err != nil {
		if err != api.ErrNotFound {
			log.Printf("Failed to get release group ID for MBID %s: %v", req.Mbid, err)
		}
		return nil, api.ErrNotFound
	}

	cacheKey := "album_review_" + releaseGroupID
	if cached, err := cacheService.GetString(ctx, &cache.GetRequest{Key: cacheKey}); err == nil && cached.Exists {
		if cached.Value == "" {
			return nil, api.ErrNotFound
		}
		return &api.AlbumInfoResponse{
			Info: &api.AlbumInfo{
				Mbid:        req.Mbid,
				Name:        req.Name,
				Description: cached.Value,
				Url:         "https://critiquebrainz.org/release-group/" + releaseGroupID,
			},
		}, nil
	} else if err != nil {
		log.Printf("Error reading album review from cache: %v", err)
	}

	description, err := fetchReviews(ctx, releaseGroupID)
	if err != nil {
		if err == api.ErrNotFound {
			if _, cacheErr := cacheService.SetString(ctx, &cache.SetStringRequest{
				Key:        cacheKey,
				Value:      "",
				TtlSeconds: int64(cacheTTL.Seconds()),
			}); cacheErr != nil {
				log.Printf("Failed to cache empty album review: %v", cacheErr)
			}
		}
		return nil, err
	}

	if _, cacheErr := cacheService.SetString(ctx, &cache.SetStringRequest{
		Key:        cacheKey,
		Value:      description,
		TtlSeconds: int64(cacheTTL.Seconds()),
	}); cacheErr != nil {
		log.Printf("Failed to cache album review: %v", cacheErr)
	}

	return &api.AlbumInfoResponse{
		Info: &api.AlbumInfo{
			Mbid:        req.Mbid,
			Name:        req.Name,
			Description: description,
			Url:         "https://critiquebrainz.org/release-group/" + releaseGroupID,
		},
	}, nil
}

func fetchReviews(ctx context.Context, releaseGroupID string) (string, error) {
	params := url.Values{}
	params.Set("entity_id", releaseGroupID)
	params.Set("entity_type", "release_group")
	params.Set("limit", "5")
	params.Set("sort", "popularity")
	params.Set("sort_order", "desc")

	resp, err := client.Get(ctx, &http.HttpRequest{
		Url:       critiquebrainzBaseURL + "/review/?" + params.Encode(),
		Headers:   map[string]string{"Accept": "application/json", "User-Agent": userAgent},
		TimeoutMs: requestTimeoutMs,
	})
	if err != nil {
		log.Printf("Error fetching reviews: %v", err)
		return "", err
	}
	if resp.Status != 200 {
		log.Printf("HTTP error: status %d", resp.Status)
		return "", fmt.Errorf("HTTP %d", resp.Status)
	}

	var reviewResp reviewResponse
	if err := json.Unmarshal(resp.Body, &reviewResp); err != nil {
		return "", err
	}
	if len(reviewResp.Reviews) == 0 {
		return "", api.ErrNotFound
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
			fmt.Fprintf(&reviewSb, "★ %d/5 ", *r.Rating)
		}
		fmt.Fprintf(&reviewSb, "by %s\n\n%s", r.User.DisplayName, r.Text)
		reviewStrings = append(reviewStrings, reviewSb.String())
	}
	sb.WriteString(strings.Join(reviewStrings, "\n---\n\n"))

	return sb.String()
}

func (CritiqueBrainzAgent) GetArtistMBID(context.Context, *api.ArtistMBIDRequest) (*api.ArtistMBIDResponse, error) {
	return nil, api.ErrNotImplemented
}

func (CritiqueBrainzAgent) GetArtistURL(context.Context, *api.ArtistURLRequest) (*api.ArtistURLResponse, error) {
	return nil, api.ErrNotImplemented
}

func (CritiqueBrainzAgent) GetArtistBiography(context.Context, *api.ArtistBiographyRequest) (*api.ArtistBiographyResponse, error) {
	return nil, api.ErrNotImplemented
}

func (CritiqueBrainzAgent) GetSimilarArtists(context.Context, *api.ArtistSimilarRequest) (*api.ArtistSimilarResponse, error) {
	return nil, api.ErrNotImplemented
}

func (CritiqueBrainzAgent) GetArtistImages(context.Context, *api.ArtistImageRequest) (*api.ArtistImageResponse, error) {
	return nil, api.ErrNotImplemented
}

func (CritiqueBrainzAgent) GetArtistTopSongs(context.Context, *api.ArtistTopSongsRequest) (*api.ArtistTopSongsResponse, error) {
	return nil, api.ErrNotImplemented
}

func (CritiqueBrainzAgent) GetAlbumImages(context.Context, *api.AlbumImagesRequest) (*api.AlbumImagesResponse, error) {
	return nil, api.ErrNotImplemented
}

func main() {}

func init() {
	log.SetFlags(0)
	log.SetPrefix("[CritiqueBrainz] ")
	api.RegisterMetadataAgent(CritiqueBrainzAgent{})
}

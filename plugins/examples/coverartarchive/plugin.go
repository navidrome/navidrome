//go:build wasip1

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/http"
)

type CoverArtArchiveAgent struct{}

var ErrNotFound = api.ErrNotFound

type caaImage struct {
	Image      string            `json:"image"`
	Front      bool              `json:"front"`
	Types      []string          `json:"types"`
	Thumbnails map[string]string `json:"thumbnails"`
}

var client = http.NewHttpService()

func (CoverArtArchiveAgent) GetAlbumImages(ctx context.Context, req *api.AlbumImagesRequest) (*api.AlbumImagesResponse, error) {
	if req.Mbid == "" {
		return nil, ErrNotFound
	}

	url := "https://coverartarchive.org/release/" + req.Mbid
	resp, err := client.Get(ctx, &http.HttpRequest{Url: url, TimeoutMs: 5000})
	if err != nil || resp.Status != 200 {
		log.Printf("[CAA] Error getting album images from CoverArtArchive (status: %d): %v", resp.Status, err)
		return nil, ErrNotFound
	}

	images, err := extractFrontImages(resp.Body)
	if err != nil || len(images) == 0 {
		return nil, ErrNotFound
	}
	return &api.AlbumImagesResponse{Images: images}, nil
}

func extractFrontImages(body []byte) ([]*api.ExternalImage, error) {
	var data struct {
		Images []caaImage `json:"images"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	img := findFrontImage(data.Images)
	if img == nil {
		return nil, ErrNotFound
	}
	return buildImageList(img), nil
}

func findFrontImage(images []caaImage) *caaImage {
	for i, img := range images {
		if img.Front {
			return &images[i]
		}
	}
	for i, img := range images {
		for _, t := range img.Types {
			if t == "Front" {
				return &images[i]
			}
		}
	}
	if len(images) > 0 {
		return &images[0]
	}
	return nil
}

func buildImageList(img *caaImage) []*api.ExternalImage {
	var images []*api.ExternalImage
	// First, try numeric sizes only
	for sizeStr, url := range img.Thumbnails {
		if url == "" {
			continue
		}
		size := 0
		if _, err := fmt.Sscanf(sizeStr, "%d", &size); err == nil {
			images = append(images, &api.ExternalImage{Url: url, Size: int32(size)})
		}
	}
	// If no numeric sizes, fallback to large/small
	if len(images) == 0 {
		for sizeStr, url := range img.Thumbnails {
			if url == "" {
				continue
			}
			var size int
			switch sizeStr {
			case "large":
				size = 500
			case "small":
				size = 250
			default:
				continue
			}
			images = append(images, &api.ExternalImage{Url: url, Size: int32(size)})
		}
	}
	if len(images) == 0 && img.Image != "" {
		images = append(images, &api.ExternalImage{Url: img.Image, Size: 0})
	}
	return images
}

func (CoverArtArchiveAgent) GetAlbumInfo(ctx context.Context, req *api.AlbumInfoRequest) (*api.AlbumInfoResponse, error) {
	return nil, api.ErrNotImplemented
}
func (CoverArtArchiveAgent) GetArtistMBID(ctx context.Context, req *api.ArtistMBIDRequest) (*api.ArtistMBIDResponse, error) {
	return nil, api.ErrNotImplemented
}

func (CoverArtArchiveAgent) GetArtistURL(ctx context.Context, req *api.ArtistURLRequest) (*api.ArtistURLResponse, error) {
	return nil, api.ErrNotImplemented
}

func (CoverArtArchiveAgent) GetArtistBiography(ctx context.Context, req *api.ArtistBiographyRequest) (*api.ArtistBiographyResponse, error) {
	return nil, api.ErrNotImplemented
}

func (CoverArtArchiveAgent) GetSimilarArtists(ctx context.Context, req *api.ArtistSimilarRequest) (*api.ArtistSimilarResponse, error) {
	return nil, api.ErrNotImplemented
}

func (CoverArtArchiveAgent) GetArtistImages(ctx context.Context, req *api.ArtistImageRequest) (*api.ArtistImageResponse, error) {
	return nil, api.ErrNotImplemented
}

func (CoverArtArchiveAgent) GetArtistTopSongs(ctx context.Context, req *api.ArtistTopSongsRequest) (*api.ArtistTopSongsResponse, error) {
	return nil, api.ErrNotImplemented
}

func main() {}

func init() {
	// Configure logging: No timestamps, no source file/line
	log.SetFlags(0)
	log.SetPrefix("[CAA] ")

	api.RegisterMetadataAgent(CoverArtArchiveAgent{})
}

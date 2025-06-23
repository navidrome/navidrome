//go:build wasip1

package main

import (
	"context"

	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/http"
)

type UnauthorizedPlugin struct{}

var ErrNotFound = api.ErrNotFound

func (UnauthorizedPlugin) GetAlbumInfo(ctx context.Context, req *api.AlbumInfoRequest) (*api.AlbumInfoResponse, error) {
	// This plugin attempts to make an HTTP call without having HTTP permission
	// This should fail since the plugin has no permissions in its manifest
	httpClient := http.NewHttpService()

	request := &http.HttpRequest{
		Url: "https://example.com/test",
		Headers: map[string]string{
			"Accept": "application/json",
		},
		TimeoutMs: 5000,
	}

	_, err := httpClient.Get(ctx, request)
	if err != nil {
		// Expected to fail due to missing permission
		return nil, err
	}

	return &api.AlbumInfoResponse{
		Info: &api.AlbumInfo{
			Name:        req.Name,
			Mbid:        "unauthorized-test",
			Description: "This should not work",
			Url:         "https://example.com/unauthorized",
		},
	}, nil
}

func (UnauthorizedPlugin) GetAlbumImages(ctx context.Context, req *api.AlbumImagesRequest) (*api.AlbumImagesResponse, error) {
	return nil, api.ErrNotImplemented
}

func (UnauthorizedPlugin) GetArtistMBID(ctx context.Context, req *api.ArtistMBIDRequest) (*api.ArtistMBIDResponse, error) {
	return nil, api.ErrNotImplemented
}

func (UnauthorizedPlugin) GetArtistURL(ctx context.Context, req *api.ArtistURLRequest) (*api.ArtistURLResponse, error) {
	return nil, api.ErrNotImplemented
}

func (UnauthorizedPlugin) GetArtistBiography(ctx context.Context, req *api.ArtistBiographyRequest) (*api.ArtistBiographyResponse, error) {
	return nil, api.ErrNotImplemented
}

func (UnauthorizedPlugin) GetSimilarArtists(ctx context.Context, req *api.ArtistSimilarRequest) (*api.ArtistSimilarResponse, error) {
	return nil, api.ErrNotImplemented
}

func (UnauthorizedPlugin) GetArtistImages(ctx context.Context, req *api.ArtistImageRequest) (*api.ArtistImageResponse, error) {
	return nil, api.ErrNotImplemented
}

func (UnauthorizedPlugin) GetArtistTopSongs(ctx context.Context, req *api.ArtistTopSongsRequest) (*api.ArtistTopSongsResponse, error) {
	return nil, api.ErrNotImplemented
}

func main() {}

// Register the plugin implementation
func init() {
	api.RegisterMetadataAgent(UnauthorizedPlugin{})
}

//go:build wasip1

package main

import (
	"context"

	"github.com/navidrome/navidrome/plugins/api"
)

type FakeAlbumAgent struct{}

var ErrNotFound = api.ErrNotFound

func (FakeAlbumAgent) GetAlbumInfo(ctx context.Context, req *api.AlbumInfoRequest) (*api.AlbumInfoResponse, error) {
	if req.Name != "" && req.Artist != "" {
		return &api.AlbumInfoResponse{
			Info: &api.AlbumInfo{
				Name:        req.Name,
				Mbid:        "album-mbid-123",
				Description: "This is a test album description",
				Url:         "https://example.com/album",
			},
		}, nil
	}
	return nil, ErrNotFound
}

func (FakeAlbumAgent) GetAlbumImages(ctx context.Context, req *api.AlbumImagesRequest) (*api.AlbumImagesResponse, error) {
	if req.Name != "" && req.Artist != "" {
		return &api.AlbumImagesResponse{
			Images: []*api.ExternalImage{
				{Url: "https://example.com/album1.jpg", Size: 300},
				{Url: "https://example.com/album2.jpg", Size: 400},
			},
		}, nil
	}
	return nil, ErrNotFound
}

func (FakeAlbumAgent) GetArtistMBID(ctx context.Context, req *api.ArtistMBIDRequest) (*api.ArtistMBIDResponse, error) {
	return nil, api.ErrNotImplemented
}

func (FakeAlbumAgent) GetArtistURL(ctx context.Context, req *api.ArtistURLRequest) (*api.ArtistURLResponse, error) {
	return nil, api.ErrNotImplemented
}

func (FakeAlbumAgent) GetArtistBiography(ctx context.Context, req *api.ArtistBiographyRequest) (*api.ArtistBiographyResponse, error) {
	return nil, api.ErrNotImplemented
}

func (FakeAlbumAgent) GetSimilarArtists(ctx context.Context, req *api.ArtistSimilarRequest) (*api.ArtistSimilarResponse, error) {
	return nil, api.ErrNotImplemented
}

func (FakeAlbumAgent) GetArtistImages(ctx context.Context, req *api.ArtistImageRequest) (*api.ArtistImageResponse, error) {
	return nil, api.ErrNotImplemented
}

func (FakeAlbumAgent) GetArtistTopSongs(ctx context.Context, req *api.ArtistTopSongsRequest) (*api.ArtistTopSongsResponse, error) {
	return nil, api.ErrNotImplemented
}

func main() {}

// Register the plugin implementation
func init() {
	api.RegisterMetadataAgent(FakeAlbumAgent{})
}

//go:build wasip1

package main

import (
	"context"

	"github.com/navidrome/navidrome/plugins/api"
)

type FakeArtistAgent struct{}

var ErrNotFound = api.ErrNotFound

func (FakeArtistAgent) GetArtistMBID(ctx context.Context, req *api.ArtistMBIDRequest) (*api.ArtistMBIDResponse, error) {
	if req.Name != "" {
		return &api.ArtistMBIDResponse{Mbid: "1234567890"}, nil
	}
	return nil, ErrNotFound
}
func (FakeArtistAgent) GetArtistURL(ctx context.Context, req *api.ArtistURLRequest) (*api.ArtistURLResponse, error) {
	if req.Name != "" {
		return &api.ArtistURLResponse{Url: "https://example.com"}, nil
	}
	return nil, ErrNotFound
}
func (FakeArtistAgent) GetArtistBiography(ctx context.Context, req *api.ArtistBiographyRequest) (*api.ArtistBiographyResponse, error) {
	if req.Name != "" {
		return &api.ArtistBiographyResponse{Biography: "This is a test biography"}, nil
	}
	return nil, ErrNotFound
}
func (FakeArtistAgent) GetSimilarArtists(ctx context.Context, req *api.ArtistSimilarRequest) (*api.ArtistSimilarResponse, error) {
	if req.Name != "" {
		return &api.ArtistSimilarResponse{
			Artists: []*api.Artist{
				{Name: "Similar Artist 1", Mbid: "mbid1"},
				{Name: "Similar Artist 2", Mbid: "mbid2"},
			},
		}, nil
	}
	return nil, ErrNotFound
}
func (FakeArtistAgent) GetArtistImages(ctx context.Context, req *api.ArtistImageRequest) (*api.ArtistImageResponse, error) {
	if req.Name != "" {
		return &api.ArtistImageResponse{
			Images: []*api.ExternalImage{
				{Url: "https://example.com/image1.jpg", Size: 100},
				{Url: "https://example.com/image2.jpg", Size: 200},
			},
		}, nil
	}
	return nil, ErrNotFound
}
func (FakeArtistAgent) GetArtistTopSongs(ctx context.Context, req *api.ArtistTopSongsRequest) (*api.ArtistTopSongsResponse, error) {
	if req.ArtistName != "" {
		return &api.ArtistTopSongsResponse{
			Songs: []*api.Song{
				{Name: "Song 1", Mbid: "mbid1"},
				{Name: "Song 2", Mbid: "mbid2"},
			},
		}, nil
	}
	return nil, ErrNotFound
}

// Add empty implementations for the album methods to satisfy the MetadataAgent interface
func (FakeArtistAgent) GetAlbumInfo(ctx context.Context, req *api.AlbumInfoRequest) (*api.AlbumInfoResponse, error) {
	return nil, api.ErrNotImplemented
}

func (FakeArtistAgent) GetAlbumImages(ctx context.Context, req *api.AlbumImagesRequest) (*api.AlbumImagesResponse, error) {
	return nil, api.ErrNotImplemented
}

// main is required by Go WASI build
func main() {}

// init is used by go-plugin to register the implementation
func init() {
	api.RegisterMetadataAgent(FakeArtistAgent{})
}

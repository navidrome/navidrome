//go:build wasip1

package main

import (
	"context"
	"log"

	"github.com/navidrome/navidrome/plugins/api"
)

type FakeMultiAgent struct{}

var ErrNotFound = api.ErrNotFound

// --- ArtistMetadataService ---
func (FakeMultiAgent) GetArtistMBID(ctx context.Context, req *api.ArtistMBIDRequest) (*api.ArtistMBIDResponse, error) {
	log.Println("FakeMultiAgent.GetArtistMBID called", req.Name)
	if req.Name != "" {
		return &api.ArtistMBIDResponse{Mbid: "multi-artist-mbid"}, nil
	}
	return nil, ErrNotFound
}
func (FakeMultiAgent) GetArtistURL(ctx context.Context, req *api.ArtistURLRequest) (*api.ArtistURLResponse, error) {
	return &api.ArtistURLResponse{Url: "https://multi.example.com/artist"}, nil
}
func (FakeMultiAgent) GetArtistBiography(ctx context.Context, req *api.ArtistBiographyRequest) (*api.ArtistBiographyResponse, error) {
	return &api.ArtistBiographyResponse{Biography: "Multi agent artist bio"}, nil
}
func (FakeMultiAgent) GetSimilarArtists(ctx context.Context, req *api.ArtistSimilarRequest) (*api.ArtistSimilarResponse, error) {
	return &api.ArtistSimilarResponse{}, nil
}
func (FakeMultiAgent) GetArtistImages(ctx context.Context, req *api.ArtistImageRequest) (*api.ArtistImageResponse, error) {
	return &api.ArtistImageResponse{}, nil
}
func (FakeMultiAgent) GetArtistTopSongs(ctx context.Context, req *api.ArtistTopSongsRequest) (*api.ArtistTopSongsResponse, error) {
	return &api.ArtistTopSongsResponse{}, nil
}

// --- AlbumMetadataService ---
func (FakeMultiAgent) GetAlbumInfo(ctx context.Context, req *api.AlbumInfoRequest) (*api.AlbumInfoResponse, error) {
	log.Println("FakeMultiAgent.GetAlbumInfo called", req.Name)
	if req.Name != "" && req.Artist != "" {
		return &api.AlbumInfoResponse{
			Info: &api.AlbumInfo{
				Name:        req.Name,
				Mbid:        "multi-album-mbid",
				Description: "Multi agent album description",
				Url:         "https://multi.example.com/album",
			},
		}, nil
	}
	return nil, ErrNotFound
}
func (FakeMultiAgent) GetAlbumImages(ctx context.Context, req *api.AlbumImagesRequest) (*api.AlbumImagesResponse, error) {
	return &api.AlbumImagesResponse{}, nil
}

func main() {}

func init() {
	api.RegisterArtistMetadataService(FakeMultiAgent{})
	api.RegisterAlbumMetadataService(FakeMultiAgent{})
}

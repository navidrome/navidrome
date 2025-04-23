//go:build wasip1

package main

import (
	"context"
	"log"

	"github.com/navidrome/navidrome/plugins/api"
)

type FakeArtistAgent struct{}

var ErrNotFound = api.ErrNotFound

func (FakeArtistAgent) GetArtistMBID(ctx context.Context, req *api.ArtistMBIDRequest) (*api.ArtistMBIDResponse, error) {
	log.Println("FakeArtistAgent.GetArtistMBID called", "id:", req.Id, "name:", req.Name)
	if req.Name != "" {
		return &api.ArtistMBIDResponse{
			Mbid: "1234567890",
		}, nil
	}
	return nil, ErrNotFound
}
func (FakeArtistAgent) GetArtistURL(ctx context.Context, req *api.ArtistURLRequest) (*api.ArtistURLResponse, error) {
	log.Println("FakeArtistAgent.GetArtistURL called", "id:", req.Id, "name:", req.Name, "mbid:", req.Mbid)
	if req.Name != "" {
		return &api.ArtistURLResponse{
			Url: "https://example.com",
		}, nil
	}
	return nil, ErrNotFound
}
func (FakeArtistAgent) GetArtistBiography(ctx context.Context, req *api.ArtistBiographyRequest) (*api.ArtistBiographyResponse, error) {
	log.Println("FakeArtistAgent.GetArtistBiography called", "id:", req.Id, "name:", req.Name, "mbid:", req.Mbid)
	if req.Name != "" {
		return &api.ArtistBiographyResponse{
			Biography: "This is a test biography",
		}, nil
	}
	return nil, ErrNotFound
}
func (FakeArtistAgent) GetSimilarArtists(ctx context.Context, req *api.ArtistSimilarRequest) (*api.ArtistSimilarResponse, error) {
	log.Println("FakeArtistAgent.GetSimilarArtists called", "id:", req.Id, "name:", req.Name, "mbid:", req.Mbid, "limit:", req.Limit)
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
	log.Println("FakeArtistAgent.GetArtistImages called", "id:", req.Id, "name:", req.Name, "mbid:", req.Mbid)
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
	log.Println("FakeArtistAgent.GetArtistTopSongs called", "id:", req.Id, "artistName:", req.ArtistName, "mbid:", req.Mbid, "count:", req.Count)
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

// main is required by Go WASI build
func main() {}

// init is used by go-plugin to register the implementation
func init() {
	api.RegisterArtistMetadataService(FakeArtistAgent{})
}

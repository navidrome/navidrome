//go:build wasip1

package main

import (
	"context"
	"log"

	"github.com/navidrome/navidrome/plugins/api"
)

type FakeAlbumAgent struct{}

var ErrNotFound = api.ErrNotFound

func (FakeAlbumAgent) GetAlbumInfo(ctx context.Context, req *api.AlbumInfoRequest) (*api.AlbumInfoResponse, error) {
	log.Println("FakeAlbumAgent.GetAlbumInfo called", "name:", req.Name, "artist:", req.Artist, "mbid:", req.Mbid)
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
	log.Println("FakeAlbumAgent.GetAlbumImages called", "name:", req.Name, "artist:", req.Artist, "mbid:", req.Mbid)
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

func main() {}

func init() {
	api.RegisterAlbumMetadataService(FakeAlbumAgent{})
}

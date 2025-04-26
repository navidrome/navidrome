package main

import (
	"context"
	"log"

	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/timer"
)

// MultiPlugin implements the MediaMetadataService interface for testing
type MultiPlugin struct{}

var ErrNotFound = api.ErrNotFound

// Artist-related methods
func (MultiPlugin) GetArtistMBID(ctx context.Context, req *api.ArtistMBIDRequest) (*api.ArtistMBIDResponse, error) {
	if req.Name != "" {
		return &api.ArtistMBIDResponse{Mbid: "multi-artist-mbid"}, nil
	}
	return nil, ErrNotFound
}

func (MultiPlugin) GetArtistURL(ctx context.Context, req *api.ArtistURLRequest) (*api.ArtistURLResponse, error) {
	log.Printf("GetArtistURL received: %v", req)

	var tmr = timer.NewTimerService()
	resp, err := tmr.RegisterTimer(ctx, &timer.TimerRequest{
		PluginName: "multi-plugin",
		Delay:      5000,
		Payload:    []byte("test-payload"),
	})
	if err != nil {
		log.Printf("Error registering timer: %v", err)
	} else {
		log.Printf("Timer registered: %v", resp)
	}
	return &api.ArtistURLResponse{Url: "https://multi.example.com/artist"}, nil
}

func (MultiPlugin) GetArtistBiography(ctx context.Context, req *api.ArtistBiographyRequest) (*api.ArtistBiographyResponse, error) {
	return &api.ArtistBiographyResponse{Biography: "Multi agent artist bio"}, nil
}

func (MultiPlugin) GetSimilarArtists(ctx context.Context, req *api.ArtistSimilarRequest) (*api.ArtistSimilarResponse, error) {
	return &api.ArtistSimilarResponse{}, nil
}

func (MultiPlugin) GetArtistImages(ctx context.Context, req *api.ArtistImageRequest) (*api.ArtistImageResponse, error) {
	return &api.ArtistImageResponse{}, nil
}

func (MultiPlugin) GetArtistTopSongs(ctx context.Context, req *api.ArtistTopSongsRequest) (*api.ArtistTopSongsResponse, error) {
	return &api.ArtistTopSongsResponse{}, nil
}

// Album-related methods
func (MultiPlugin) GetAlbumInfo(ctx context.Context, req *api.AlbumInfoRequest) (*api.AlbumInfoResponse, error) {
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

func (MultiPlugin) GetAlbumImages(ctx context.Context, req *api.AlbumImagesRequest) (*api.AlbumImagesResponse, error) {
	return &api.AlbumImagesResponse{}, nil
}

// Timer-related methods
func (MultiPlugin) OnTimerCallback(ctx context.Context, req *api.TimerCallbackRequest) (*api.TimerCallbackResponse, error) {
	log.Printf("Timer callback received: %v", req)
	return &api.TimerCallbackResponse{}, nil
}

//go:build wasip1

package main

import (
	"context"
	"log"
	"strings"

	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/timer"
)

// MultiPlugin implements the MetadataAgent interface for testing
type MultiPlugin struct{}

var ErrNotFound = api.ErrNotFound

var tmr = timer.NewTimerService()

// Artist-related methods
func (MultiPlugin) GetArtistMBID(ctx context.Context, req *api.ArtistMBIDRequest) (*api.ArtistMBIDResponse, error) {
	if req.Name != "" {
		return &api.ArtistMBIDResponse{Mbid: "multi-artist-mbid"}, nil
	}
	return nil, ErrNotFound
}

func (MultiPlugin) GetArtistURL(ctx context.Context, req *api.ArtistURLRequest) (*api.ArtistURLResponse, error) {
	log.Printf("GetArtistURL received: %v", req)

	// Use an ID that could potentially clash with other plugins
	// The host will ensure this doesn't conflict by prefixing with plugin name
	customTimerId := "artist:" + req.Name
	log.Printf("Registering timer with custom ID: %s", customTimerId)

	resp, err := tmr.RegisterTimer(ctx, &timer.TimerRequest{
		TimerId: customTimerId,
		Delay:   6,
		Payload: []byte("test-payload"),
	})
	if err != nil {
		log.Printf("Error registering timer: %v", err)
	} else {
		log.Printf("Timer registered with ID: %s", resp.TimerId)
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
	log.Printf("Timer callback received with ID: %s, payload: '%s'", req.TimerId, string(req.Payload))

	// Demonstrate how to parse the custom timer ID format
	if strings.HasPrefix(req.TimerId, "artist:") {
		parts := strings.Split(req.TimerId, ":")
		if len(parts) == 2 {
			artistName := parts[1]
			log.Printf("This timer was for artist: %s", artistName)
		}
	}

	return &api.TimerCallbackResponse{}, nil
}

func (MultiPlugin) OnInit(ctx context.Context, req *api.InitRequest) (*api.InitResponse, error) {
	log.Printf("OnInit called with %v", req)
	_, _ = tmr.RegisterTimer(ctx, &timer.TimerRequest{
		Delay:   2,
		Payload: []byte("2 seconds after init"),
	})

	return &api.InitResponse{}, nil
}

// Required by Go WASI build
func main() {}

// Register the service implementations
func init() {
	api.RegisterMetadataAgent(MultiPlugin{})
	api.RegisterTimerCallback(MultiPlugin{})
	api.RegisterLifecycleManagement(MultiPlugin{})
}

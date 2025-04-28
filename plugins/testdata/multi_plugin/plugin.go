//go:build wasip1

package main

import (
	"context"
	"log"
	"strings"

	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/scheduler"
)

// MultiPlugin implements the MetadataAgent interface for testing
type MultiPlugin struct{}

var ErrNotFound = api.ErrNotFound

var sched = scheduler.NewSchedulerService()

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
	customId := "artist:" + req.Name
	log.Printf("Registering scheduler with custom ID: %s", customId)

	// Use the scheduler service for one-time scheduling
	resp, err := sched.ScheduleOneTime(ctx, &scheduler.ScheduleOneTimeRequest{
		ScheduleId:   customId,
		DelaySeconds: 6,
		Payload:      []byte("test-payload"),
	})
	if err != nil {
		log.Printf("Error scheduling one-time job: %v", err)
	} else {
		log.Printf("One-time schedule registered with ID: %s", resp.ScheduleId)
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

// Scheduler callback
func (MultiPlugin) OnSchedulerCallback(ctx context.Context, req *api.SchedulerCallbackRequest) (*api.SchedulerCallbackResponse, error) {
	log.Printf("Scheduler callback received with ID: %s, payload: '%s', isRecurring: %v",
		req.ScheduleId, string(req.Payload), req.IsRecurring)

	// Demonstrate how to parse the custom ID format
	if strings.HasPrefix(req.ScheduleId, "artist:") {
		parts := strings.Split(req.ScheduleId, ":")
		if len(parts) == 2 {
			artistName := parts[1]
			log.Printf("This schedule was for artist: %s", artistName)
		}
	}

	return &api.SchedulerCallbackResponse{}, nil
}

func (MultiPlugin) OnInit(ctx context.Context, req *api.InitRequest) (*api.InitResponse, error) {
	log.Printf("OnInit called with %v", req)

	// Schedule a recurring every 5 seconds
	_, _ = sched.ScheduleRecurring(ctx, &scheduler.ScheduleRecurringRequest{
		CronExpression: "@every 5s",
		Payload:        []byte("every 5 seconds"),
	})

	return &api.InitResponse{}, nil
}

// Required by Go WASI build
func main() {}

// Register the service implementations
func init() {
	api.RegisterLifecycleManagement(MultiPlugin{})
	api.RegisterMetadataAgent(MultiPlugin{})
	api.RegisterSchedulerCallback(MultiPlugin{})
}

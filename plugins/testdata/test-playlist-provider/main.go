// Test playlist provider plugin for Navidrome plugin system integration tests.
package main

import (
	"fmt"
	"strconv"

	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	pp "github.com/navidrome/navidrome/plugins/pdk/go/playlistprovider"
)

func init() {
	pp.Register(&testPlaylistProvider{})
}

type testPlaylistProvider struct{}

func (t *testPlaylistProvider) GetAvailablePlaylists(_ pp.GetAvailablePlaylistsRequest) (pp.GetAvailablePlaylistsResponse, error) {
	// Check for configured error
	errMsg, hasErr := pdk.GetConfig("error")
	if hasErr && errMsg != "" {
		return pp.GetAvailablePlaylistsResponse{}, fmt.Errorf("%s", errMsg)
	}

	// Get the owner username from config (defaults to "admin")
	ownerUsername := "admin"
	if u, ok := pdk.GetConfig("owner_username"); ok && u != "" {
		ownerUsername = u
	}

	resp := pp.GetAvailablePlaylistsResponse{
		Playlists: []pp.PlaylistInfo{
			{ID: "daily-mix-1", OwnerUsername: ownerUsername},
			{ID: "daily-mix-2", OwnerUsername: ownerUsername},
		},
		RefreshInterval: 0, // No re-discovery in tests
	}

	// Support configurable retry interval
	if ri, ok := pdk.GetConfig("retry_interval"); ok && ri != "" {
		if v, err := strconv.ParseInt(ri, 10, 64); err == nil {
			resp.RetryInterval = v
		}
	}

	return resp, nil
}

func (t *testPlaylistProvider) GetPlaylist(req pp.GetPlaylistRequest) (pp.GetPlaylistResponse, error) {
	// Check for configured error
	errMsg, hasErr := pdk.GetConfig("get_playlist_error")
	if hasErr && errMsg != "" {
		// Check if the error should be typed (e.g., NotFound)
		errType, _ := pdk.GetConfig("get_playlist_error_type")
		if errType == pp.PlaylistProviderErrorNotFound.Error() {
			return pp.GetPlaylistResponse{}, fmt.Errorf("%w: %s", pp.PlaylistProviderErrorNotFound, errMsg)
		}
		return pp.GetPlaylistResponse{}, fmt.Errorf("%s", errMsg)
	}

	switch req.ID {
	case "daily-mix-1":
		return pp.GetPlaylistResponse{
			Name:        "Daily Mix 1",
			Description: "Your personalized daily mix",
			CoverArtURL: "https://example.com/cover1.jpg",
			Tracks: []pp.SongRef{
				{Name: "Song A", Artist: "Artist One"},
				{Name: "Song B", Artist: "Artist Two"},
			},
			ValidUntil: 0, // Static, no refresh
		}, nil
	case "daily-mix-2":
		return pp.GetPlaylistResponse{
			Name: "Daily Mix 2",
			Tracks: []pp.SongRef{
				{Name: "Song C", Artist: "Artist Three"},
			},
			ValidUntil: 0,
		}, nil
	default:
		return pp.GetPlaylistResponse{}, fmt.Errorf("unknown playlist: %s", req.ID)
	}
}

func main() {}

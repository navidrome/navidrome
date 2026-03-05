// Test playlist generator plugin for Navidrome plugin system integration tests.
package main

import (
	"fmt"

	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	pg "github.com/navidrome/navidrome/plugins/pdk/go/playlistgenerator"
)

func init() {
	pg.Register(&testPlaylistGenerator{})
}

type testPlaylistGenerator struct{}

func (t *testPlaylistGenerator) GetPlaylists(_ pg.GetPlaylistsRequest) (pg.GetPlaylistsResponse, error) {
	// Check for configured error
	errMsg, hasErr := pdk.GetConfig("error")
	if hasErr && errMsg != "" {
		return pg.GetPlaylistsResponse{}, fmt.Errorf("%s", errMsg)
	}

	// Get the owner username from config (defaults to "admin")
	ownerUsername := "admin"
	if u, ok := pdk.GetConfig("owner_username"); ok && u != "" {
		ownerUsername = u
	}

	return pg.GetPlaylistsResponse{
		Playlists: []pg.PlaylistInfo{
			{ID: "daily-mix-1", OwnerUsername: ownerUsername},
			{ID: "daily-mix-2", OwnerUsername: ownerUsername},
		},
		RefreshInterval: 0, // No re-discovery in tests
	}, nil
}

func (t *testPlaylistGenerator) GetPlaylist(req pg.GetPlaylistRequest) (pg.GetPlaylistResponse, error) {
	// Check for configured error
	errMsg, hasErr := pdk.GetConfig("get_playlist_error")
	if hasErr && errMsg != "" {
		return pg.GetPlaylistResponse{}, fmt.Errorf("%s", errMsg)
	}

	switch req.ID {
	case "daily-mix-1":
		return pg.GetPlaylistResponse{
			Name:        "Daily Mix 1",
			Description: "Your personalized daily mix",
			CoverArtURL: "https://example.com/cover1.jpg",
			Tracks: []pg.SongRef{
				{Name: "Song A", Artist: "Artist One"},
				{Name: "Song B", Artist: "Artist Two"},
			},
			ValidUntil: 0, // Static, no refresh
		}, nil
	case "daily-mix-2":
		return pg.GetPlaylistResponse{
			Name: "Daily Mix 2",
			Tracks: []pg.SongRef{
				{Name: "Song C", Artist: "Artist Three"},
			},
			ValidUntil: 0,
		}, nil
	default:
		return pg.GetPlaylistResponse{}, fmt.Errorf("unknown playlist: %s", req.ID)
	}
}

func main() {}

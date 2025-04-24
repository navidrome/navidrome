//go:build wasip1

package main

import (
	"context"
	"fmt"

	"github.com/navidrome/navidrome/plugins/api"
)

// Example implementation of the combined MediaMetadataService
// This plugin demonstrates how to implement a service that handles both
// artist and album metadata in a single plugin

type Plugin struct{}

func (p *Plugin) GetArtistMBID(ctx context.Context, req *api.ArtistMBIDRequest) (*api.ArtistMBIDResponse, error) {
	// Implementation logic here
	// You can use req.Id and req.Name to search for the MBID

	if req.Name == "" {
		return nil, api.ErrNotFound
	}

	// Return example MBID - in a real plugin, this would be fetched from a data source
	return &api.ArtistMBIDResponse{
		Mbid: fmt.Sprintf("example-artist-mbid-%s", req.Name),
	}, nil
}

func (p *Plugin) GetArtistURL(ctx context.Context, req *api.ArtistURLRequest) (*api.ArtistURLResponse, error) {
	if req.Name == "" {
		return nil, api.ErrNotFound
	}

	return &api.ArtistURLResponse{
		Url: fmt.Sprintf("https://example.com/artists/%s", req.Name),
	}, nil
}

func (p *Plugin) GetArtistBiography(ctx context.Context, req *api.ArtistBiographyRequest) (*api.ArtistBiographyResponse, error) {
	if req.Name == "" {
		return nil, api.ErrNotFound
	}

	return &api.ArtistBiographyResponse{
		Biography: fmt.Sprintf("This is a bio for %s", req.Name),
	}, nil
}

func (p *Plugin) GetSimilarArtists(ctx context.Context, req *api.ArtistSimilarRequest) (*api.ArtistSimilarResponse, error) {
	if req.Name == "" {
		return nil, api.ErrNotFound
	}

	// Create example similar artists
	artists := make([]*api.Artist, 0, req.Limit)
	for i := 0; i < int(req.Limit); i++ {
		artists = append(artists, &api.Artist{
			Name: fmt.Sprintf("Similar artist %d to %s", i+1, req.Name),
			Mbid: fmt.Sprintf("similar-artist-mbid-%d", i+1),
		})
	}

	return &api.ArtistSimilarResponse{
		Artists: artists,
	}, nil
}

func (p *Plugin) GetArtistImages(ctx context.Context, req *api.ArtistImageRequest) (*api.ArtistImageResponse, error) {
	if req.Name == "" {
		return nil, api.ErrNotFound
	}

	// Create example image sizes
	images := []*api.ExternalImage{
		{Url: fmt.Sprintf("https://example.com/images/artists/%s/small", req.Name), Size: 100},
		{Url: fmt.Sprintf("https://example.com/images/artists/%s/medium", req.Name), Size: 300},
		{Url: fmt.Sprintf("https://example.com/images/artists/%s/large", req.Name), Size: 600},
	}

	return &api.ArtistImageResponse{
		Images: images,
	}, nil
}

func (p *Plugin) GetArtistTopSongs(ctx context.Context, req *api.ArtistTopSongsRequest) (*api.ArtistTopSongsResponse, error) {
	if req.ArtistName == "" {
		return nil, api.ErrNotFound
	}

	// Create example top songs
	songs := make([]*api.Song, 0, req.Count)
	for i := 0; i < int(req.Count); i++ {
		songs = append(songs, &api.Song{
			Name: fmt.Sprintf("Top song %d by %s", i+1, req.ArtistName),
			Mbid: fmt.Sprintf("top-song-mbid-%d", i+1),
		})
	}

	return &api.ArtistTopSongsResponse{
		Songs: songs,
	}, nil
}

func (p *Plugin) GetAlbumInfo(ctx context.Context, req *api.AlbumInfoRequest) (*api.AlbumInfoResponse, error) {
	if req.Name == "" || req.Artist == "" {
		return nil, api.ErrNotFound
	}

	// Create example album info
	albumInfo := &api.AlbumInfo{
		Name:        req.Name,
		Mbid:        fmt.Sprintf("example-album-mbid-%s-%s", req.Artist, req.Name),
		Description: fmt.Sprintf("This is %s by %s", req.Name, req.Artist),
		Url:         fmt.Sprintf("https://example.com/albums/%s/%s", req.Artist, req.Name),
	}

	return &api.AlbumInfoResponse{
		Info: albumInfo,
	}, nil
}

func (p *Plugin) GetAlbumImages(ctx context.Context, req *api.AlbumImagesRequest) (*api.AlbumImagesResponse, error) {
	if req.Name == "" || req.Artist == "" {
		return nil, api.ErrNotFound
	}

	// Create example image sizes
	images := []*api.ExternalImage{
		{Url: fmt.Sprintf("https://example.com/images/albums/%s/%s/cover-small", req.Artist, req.Name), Size: 150},
		{Url: fmt.Sprintf("https://example.com/images/albums/%s/%s/cover-medium", req.Artist, req.Name), Size: 350},
		{Url: fmt.Sprintf("https://example.com/images/albums/%s/%s/cover-large", req.Artist, req.Name), Size: 700},
	}

	return &api.AlbumImagesResponse{
		Images: images,
	}, nil
}

// Main function is required but empty for Go plugins
func main() {}

func init() {
	api.RegisterMediaMetadataService(Plugin{})
}

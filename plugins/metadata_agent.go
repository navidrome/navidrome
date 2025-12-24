package plugins

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/core/agents"
)

// CapabilityMetadataAgent indicates the plugin can provide artist/album metadata.
// Detected when the plugin exports at least one of the metadata agent functions.
const CapabilityMetadataAgent Capability = "MetadataAgent"

// Export function names (snake_case as per design)
const (
	FuncGetArtistMBID      = "nd_get_artist_mbid"
	FuncGetArtistURL       = "nd_get_artist_url"
	FuncGetArtistBiography = "nd_get_artist_biography"
	FuncGetSimilarArtists  = "nd_get_similar_artists"
	FuncGetArtistImages    = "nd_get_artist_images"
	FuncGetArtistTopSongs  = "nd_get_artist_top_songs"
	FuncGetAlbumInfo       = "nd_get_album_info"
	FuncGetAlbumImages     = "nd_get_album_images"
)

func init() {
	registerCapability(
		CapabilityMetadataAgent,
		FuncGetArtistMBID,
		FuncGetArtistURL,
		FuncGetArtistBiography,
		FuncGetSimilarArtists,
		FuncGetArtistImages,
		FuncGetArtistTopSongs,
		FuncGetAlbumInfo,
		FuncGetAlbumImages,
	)
}

// MetadataAgent is an adapter that wraps an Extism plugin and implements
// the agents interfaces for metadata retrieval.
type MetadataAgent struct {
	name   string
	plugin *pluginInstance
}

// AgentName returns the plugin name
func (a *MetadataAgent) AgentName() string {
	return a.name
}

// --- Interface implementations ---

// GetArtistMBID retrieves the MusicBrainz ID for an artist
func (a *MetadataAgent) GetArtistMBID(ctx context.Context, id string, name string) (string, error) {
	input := artistMBIDInput{ID: id, Name: name}
	result, err := callPluginFunction[artistMBIDInput, artistMBIDOutput](ctx, a.plugin, FuncGetArtistMBID, input)
	if err != nil {
		return "", errors.Join(agents.ErrNotFound, err)
	}

	if result.MBID == "" {
		return "", agents.ErrNotFound
	}

	return result.MBID, nil
}

// GetArtistURL retrieves the external URL for an artist
func (a *MetadataAgent) GetArtistURL(ctx context.Context, id, name, mbid string) (string, error) {
	input := artistInput{ID: id, Name: name, MBID: mbid}
	result, err := callPluginFunction[artistInput, artistURLOutput](ctx, a.plugin, FuncGetArtistURL, input)
	if err != nil {
		return "", errors.Join(agents.ErrNotFound, err)
	}
	if result.URL == "" {
		return "", agents.ErrNotFound
	}
	return result.URL, nil
}

// GetArtistBiography retrieves the biography for an artist
func (a *MetadataAgent) GetArtistBiography(ctx context.Context, id, name, mbid string) (string, error) {
	input := artistInput{ID: id, Name: name, MBID: mbid}
	result, err := callPluginFunction[artistInput, artistBiographyOutput](ctx, a.plugin, FuncGetArtistBiography, input)
	if err != nil {
		return "", errors.Join(agents.ErrNotFound, err)
	}

	if result.Biography == "" {
		return "", agents.ErrNotFound
	}

	return result.Biography, nil
}

// GetSimilarArtists retrieves similar artists
func (a *MetadataAgent) GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]agents.Artist, error) {
	input := similarArtistsInput{ID: id, Name: name, MBID: mbid, Limit: limit}
	result, err := callPluginFunction[similarArtistsInput, similarArtistsOutput](ctx, a.plugin, FuncGetSimilarArtists, input)
	if err != nil {
		return nil, errors.Join(agents.ErrNotFound, err)
	}

	if len(result.Artists) == 0 {
		return nil, agents.ErrNotFound
	}

	artists := make([]agents.Artist, len(result.Artists))
	for i, a := range result.Artists {
		artists[i] = agents.Artist{Name: a.Name, MBID: a.MBID}
	}

	return artists, nil
}

// GetArtistImages retrieves images for an artist
func (a *MetadataAgent) GetArtistImages(ctx context.Context, id, name, mbid string) ([]agents.ExternalImage, error) {
	input := artistInput{ID: id, Name: name, MBID: mbid}
	result, err := callPluginFunction[artistInput, artistImagesOutput](ctx, a.plugin, FuncGetArtistImages, input)
	if err != nil {
		return nil, errors.Join(agents.ErrNotFound, err)
	}

	if len(result.Images) == 0 {
		return nil, agents.ErrNotFound
	}

	images := make([]agents.ExternalImage, len(result.Images))
	for i, img := range result.Images {
		images[i] = agents.ExternalImage{URL: img.URL, Size: img.Size}
	}

	return images, nil
}

// GetArtistTopSongs retrieves top songs for an artist
func (a *MetadataAgent) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error) {
	input := topSongsInput{ID: id, Name: artistName, MBID: mbid, Count: count}
	result, err := callPluginFunction[topSongsInput, topSongsOutput](ctx, a.plugin, FuncGetArtistTopSongs, input)
	if err != nil {
		return nil, errors.Join(agents.ErrNotFound, err)
	}

	if len(result.Songs) == 0 {
		return nil, agents.ErrNotFound
	}

	songs := make([]agents.Song, len(result.Songs))
	for i, s := range result.Songs {
		songs[i] = agents.Song{Name: s.Name, MBID: s.MBID}
	}

	return songs, nil
}

// GetAlbumInfo retrieves album information
func (a *MetadataAgent) GetAlbumInfo(ctx context.Context, name, artist, mbid string) (*agents.AlbumInfo, error) {
	input := albumInput{Name: name, Artist: artist, MBID: mbid}
	result, err := callPluginFunction[albumInput, albumInfoOutput](ctx, a.plugin, FuncGetAlbumInfo, input)
	if err != nil {
		return nil, errors.Join(agents.ErrNotFound, err)
	}

	return &agents.AlbumInfo{
		Name:        result.Name,
		MBID:        result.MBID,
		Description: result.Description,
		URL:         result.URL,
	}, nil
}

// GetAlbumImages retrieves images for an album
func (a *MetadataAgent) GetAlbumImages(ctx context.Context, name, artist, mbid string) ([]agents.ExternalImage, error) {
	input := albumInput{Name: name, Artist: artist, MBID: mbid}
	result, err := callPluginFunction[albumInput, albumImagesOutput](ctx, a.plugin, FuncGetAlbumImages, input)
	if err != nil {
		return nil, errors.Join(agents.ErrNotFound, err)
	}

	if len(result.Images) == 0 {
		return nil, agents.ErrNotFound
	}

	images := make([]agents.ExternalImage, len(result.Images))
	for i, img := range result.Images {
		images[i] = agents.ExternalImage{URL: img.URL, Size: img.Size}
	}

	return images, nil
}

// Verify interface implementations at compile time
var (
	_ agents.Interface                = (*MetadataAgent)(nil)
	_ agents.ArtistMBIDRetriever      = (*MetadataAgent)(nil)
	_ agents.ArtistURLRetriever       = (*MetadataAgent)(nil)
	_ agents.ArtistBiographyRetriever = (*MetadataAgent)(nil)
	_ agents.ArtistSimilarRetriever   = (*MetadataAgent)(nil)
	_ agents.ArtistImageRetriever     = (*MetadataAgent)(nil)
	_ agents.ArtistTopSongsRetriever  = (*MetadataAgent)(nil)
	_ agents.AlbumInfoRetriever       = (*MetadataAgent)(nil)
	_ agents.AlbumImageRetriever      = (*MetadataAgent)(nil)
)

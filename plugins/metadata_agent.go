package plugins

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/plugins/capabilities"
)

// CapabilityMetadataAgent indicates the plugin can provide artist/album metadata.
// Detected when the plugin exports at least one of the metadata agent functions.
const CapabilityMetadataAgent Capability = "MetadataAgent"

// Export function names (snake_case as per design)
const (
	FuncGetArtistMBID           = "nd_get_artist_mbid"
	FuncGetArtistURL            = "nd_get_artist_url"
	FuncGetArtistBiography      = "nd_get_artist_biography"
	FuncGetSimilarArtists       = "nd_get_similar_artists"
	FuncGetArtistImages         = "nd_get_artist_images"
	FuncGetArtistTopSongs       = "nd_get_artist_top_songs"
	FuncGetAlbumInfo            = "nd_get_album_info"
	FuncGetAlbumImages          = "nd_get_album_images"
	FuncGetSimilarSongsByTrack  = "nd_get_similar_songs_by_track"
	FuncGetSimilarSongsByAlbum  = "nd_get_similar_songs_by_album"
	FuncGetSimilarSongsByArtist = "nd_get_similar_songs_by_artist"
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
		FuncGetSimilarSongsByTrack,
		FuncGetSimilarSongsByAlbum,
		FuncGetSimilarSongsByArtist,
	)
}

// MetadataAgent is an adapter that wraps an Extism plugin and implements
// the agents interfaces for metadata retrieval.
type MetadataAgent struct {
	name   string
	plugin *plugin
}

// AgentName returns the plugin name
func (a *MetadataAgent) AgentName() string {
	return a.name
}

// --- Interface implementations ---

// GetArtistMBID retrieves the MusicBrainz ID for an artist
func (a *MetadataAgent) GetArtistMBID(ctx context.Context, id string, name string) (string, error) {
	input := capabilities.ArtistMBIDRequest{ID: id, Name: name}
	result, err := callPluginFunction[capabilities.ArtistMBIDRequest, *capabilities.ArtistMBIDResponse](ctx, a.plugin, FuncGetArtistMBID, input)
	if err != nil {
		return "", errors.Join(agents.ErrNotFound, err)
	}

	if result == nil || result.MBID == "" {
		return "", agents.ErrNotFound
	}

	return result.MBID, nil
}

// GetArtistURL retrieves the external URL for an artist
func (a *MetadataAgent) GetArtistURL(ctx context.Context, id, name, mbid string) (string, error) {
	input := capabilities.ArtistRequest{ID: id, Name: name, MBID: mbid}
	result, err := callPluginFunction[capabilities.ArtistRequest, *capabilities.ArtistURLResponse](ctx, a.plugin, FuncGetArtistURL, input)
	if err != nil {
		return "", errors.Join(agents.ErrNotFound, err)
	}
	if result == nil || result.URL == "" {
		return "", agents.ErrNotFound
	}
	return result.URL, nil
}

// GetArtistBiography retrieves the biography for an artist
func (a *MetadataAgent) GetArtistBiography(ctx context.Context, id, name, mbid string) (string, error) {
	input := capabilities.ArtistRequest{ID: id, Name: name, MBID: mbid}
	result, err := callPluginFunction[capabilities.ArtistRequest, *capabilities.ArtistBiographyResponse](ctx, a.plugin, FuncGetArtistBiography, input)
	if err != nil {
		return "", errors.Join(agents.ErrNotFound, err)
	}

	if result == nil || result.Biography == "" {
		return "", agents.ErrNotFound
	}

	return result.Biography, nil
}

// GetSimilarArtists retrieves similar artists
func (a *MetadataAgent) GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]agents.Artist, error) {
	input := capabilities.SimilarArtistsRequest{ID: id, Name: name, MBID: mbid, Limit: int32(limit)}
	result, err := callPluginFunction[capabilities.SimilarArtistsRequest, *capabilities.SimilarArtistsResponse](ctx, a.plugin, FuncGetSimilarArtists, input)
	if err != nil {
		return nil, errors.Join(agents.ErrNotFound, err)
	}

	if result == nil || len(result.Artists) == 0 {
		return nil, agents.ErrNotFound
	}

	artists := make([]agents.Artist, len(result.Artists))
	for i, ar := range result.Artists {
		artists[i] = agents.Artist{ID: ar.ID, Name: ar.Name, MBID: ar.MBID}
	}

	return artists, nil
}

// GetArtistImages retrieves images for an artist
func (a *MetadataAgent) GetArtistImages(ctx context.Context, id, name, mbid string) ([]agents.ExternalImage, error) {
	input := capabilities.ArtistRequest{ID: id, Name: name, MBID: mbid}
	result, err := callPluginFunction[capabilities.ArtistRequest, *capabilities.ArtistImagesResponse](ctx, a.plugin, FuncGetArtistImages, input)
	if err != nil {
		return nil, errors.Join(agents.ErrNotFound, err)
	}

	if result == nil || len(result.Images) == 0 {
		return nil, agents.ErrNotFound
	}

	images := make([]agents.ExternalImage, len(result.Images))
	for i, img := range result.Images {
		images[i] = agents.ExternalImage{URL: img.URL, Size: int(img.Size)}
	}

	return images, nil
}

// GetArtistTopSongs retrieves top songs for an artist
func (a *MetadataAgent) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error) {
	input := capabilities.TopSongsRequest{ID: id, Name: artistName, MBID: mbid, Count: int32(count)}
	result, err := callPluginFunction[capabilities.TopSongsRequest, *capabilities.TopSongsResponse](ctx, a.plugin, FuncGetArtistTopSongs, input)
	if err != nil {
		return nil, errors.Join(agents.ErrNotFound, err)
	}

	if result == nil || len(result.Songs) == 0 {
		return nil, agents.ErrNotFound
	}

	return songRefsToAgentSongs(result.Songs), nil
}

// GetAlbumInfo retrieves album information
func (a *MetadataAgent) GetAlbumInfo(ctx context.Context, name, artist, mbid string) (*agents.AlbumInfo, error) {
	input := capabilities.AlbumRequest{Name: name, Artist: artist, MBID: mbid}
	result, err := callPluginFunction[capabilities.AlbumRequest, *capabilities.AlbumInfoResponse](ctx, a.plugin, FuncGetAlbumInfo, input)
	if err != nil {
		return nil, errors.Join(agents.ErrNotFound, err)
	}

	if result == nil {
		return nil, agents.ErrNotFound
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
	input := capabilities.AlbumRequest{Name: name, Artist: artist, MBID: mbid}
	result, err := callPluginFunction[capabilities.AlbumRequest, *capabilities.AlbumImagesResponse](ctx, a.plugin, FuncGetAlbumImages, input)
	if err != nil {
		return nil, errors.Join(agents.ErrNotFound, err)
	}

	if result == nil || len(result.Images) == 0 {
		return nil, agents.ErrNotFound
	}

	images := make([]agents.ExternalImage, len(result.Images))
	for i, img := range result.Images {
		images[i] = agents.ExternalImage{URL: img.URL, Size: int(img.Size)}
	}

	return images, nil
}

func callSimilarSongsPluginFunction[T any](ctx context.Context, plugin *plugin, funcName string, input T) ([]agents.Song, error) {
	result, err := callPluginFunction[T, *capabilities.SimilarSongsResponse](ctx, plugin, funcName, input)
	if err != nil {
		return nil, err
	}
	if result == nil || len(result.Songs) == 0 {
		return nil, agents.ErrNotFound
	}
	return songRefsToAgentSongs(result.Songs), nil
}

// GetSimilarSongsByTrack retrieves songs similar to a specific track
func (a *MetadataAgent) GetSimilarSongsByTrack(ctx context.Context, id, name, artist, mbid string, count int) ([]agents.Song, error) {
	return callSimilarSongsPluginFunction[capabilities.SimilarSongsByTrackRequest](ctx, a.plugin, FuncGetSimilarSongsByTrack, capabilities.SimilarSongsByTrackRequest{ID: id, Name: name, Artist: artist, MBID: mbid, Count: int32(count)})
}

// GetSimilarSongsByAlbum retrieves songs similar to tracks on an album
func (a *MetadataAgent) GetSimilarSongsByAlbum(ctx context.Context, id, name, artist, mbid string, count int) ([]agents.Song, error) {
	return callSimilarSongsPluginFunction[capabilities.SimilarSongsByAlbumRequest](ctx, a.plugin, FuncGetSimilarSongsByAlbum, capabilities.SimilarSongsByAlbumRequest{ID: id, Name: name, Artist: artist, MBID: mbid, Count: int32(count)})
}

// GetSimilarSongsByArtist retrieves songs similar to an artist's catalog
func (a *MetadataAgent) GetSimilarSongsByArtist(ctx context.Context, id, name, mbid string, count int) ([]agents.Song, error) {
	return callSimilarSongsPluginFunction[capabilities.SimilarSongsByArtistRequest](ctx, a.plugin, FuncGetSimilarSongsByArtist, capabilities.SimilarSongsByArtistRequest{ID: id, Name: name, MBID: mbid, Count: int32(count)})
}

// songRefsToAgentSongs converts a slice of SongRef to agents.Song
func songRefsToAgentSongs(refs []capabilities.SongRef) []agents.Song {
	songs := make([]agents.Song, len(refs))
	for i, s := range refs {
		songs[i] = agents.Song{
			ID:         s.ID,
			Name:       s.Name,
			MBID:       s.MBID,
			Artist:     s.Artist,
			ArtistMBID: s.ArtistMBID,
			Album:      s.Album,
			AlbumMBID:  s.AlbumMBID,
			Duration:   uint32(s.Duration * 1000),
		}
	}
	return songs
}

// Verify interface implementations at compile time
var (
	_ agents.Interface                     = (*MetadataAgent)(nil)
	_ agents.ArtistMBIDRetriever           = (*MetadataAgent)(nil)
	_ agents.ArtistURLRetriever            = (*MetadataAgent)(nil)
	_ agents.ArtistBiographyRetriever      = (*MetadataAgent)(nil)
	_ agents.ArtistSimilarRetriever        = (*MetadataAgent)(nil)
	_ agents.ArtistImageRetriever          = (*MetadataAgent)(nil)
	_ agents.ArtistTopSongsRetriever       = (*MetadataAgent)(nil)
	_ agents.AlbumInfoRetriever            = (*MetadataAgent)(nil)
	_ agents.AlbumImageRetriever           = (*MetadataAgent)(nil)
	_ agents.SimilarSongsByTrackRetriever  = (*MetadataAgent)(nil)
	_ agents.SimilarSongsByAlbumRetriever  = (*MetadataAgent)(nil)
	_ agents.SimilarSongsByArtistRetriever = (*MetadataAgent)(nil)
)

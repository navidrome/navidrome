package plugins

import (
	"context"
	"encoding/json"

	extism "github.com/extism/go-sdk"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
)

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

// MetadataAgent is an adapter that wraps an Extism plugin and implements
// the agents interfaces for metadata retrieval.
type MetadataAgent struct {
	name   string
	plugin *extism.Plugin
}

// NewMetadataAgent creates a new MetadataAgent wrapping the given plugin.
func NewMetadataAgent(name string, plugin *extism.Plugin) *MetadataAgent {
	return &MetadataAgent{
		name:   name,
		plugin: plugin,
	}
}

// AgentName returns the plugin name
func (a *MetadataAgent) AgentName() string {
	return a.name
}

// Close closes the plugin instance
func (a *MetadataAgent) Close() error {
	if a.plugin != nil {
		return a.plugin.Close(context.Background())
	}
	return nil
}

// --- Input/Output JSON structures ---

type artistMBIDInput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type artistMBIDOutput struct {
	MBID string `json:"mbid"`
}

type artistInput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	MBID string `json:"mbid,omitempty"`
}

type artistURLOutput struct {
	URL string `json:"url"`
}

type artistBiographyOutput struct {
	Biography string `json:"biography"`
}

type similarArtistsInput struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	MBID  string `json:"mbid,omitempty"`
	Limit int    `json:"limit"`
}

type similarArtistsOutput struct {
	Artists []struct {
		Name string `json:"name"`
		MBID string `json:"mbid,omitempty"`
	} `json:"artists"`
}

type artistImagesOutput struct {
	Images []struct {
		URL  string `json:"url"`
		Size int    `json:"size"`
	} `json:"images"`
}

type topSongsInput struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	MBID  string `json:"mbid,omitempty"`
	Count int    `json:"count"`
}

type topSongsOutput struct {
	Songs []struct {
		Name string `json:"name"`
		MBID string `json:"mbid,omitempty"`
	} `json:"songs"`
}

type albumInput struct {
	Name   string `json:"name"`
	Artist string `json:"artist"`
	MBID   string `json:"mbid,omitempty"`
}

type albumInfoOutput struct {
	Name        string `json:"name"`
	MBID        string `json:"mbid"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

type albumImagesOutput struct {
	Images []struct {
		URL  string `json:"url"`
		Size int    `json:"size"`
	} `json:"images"`
}

// --- Interface implementations ---

// GetArtistMBID retrieves the MusicBrainz ID for an artist
func (a *MetadataAgent) GetArtistMBID(ctx context.Context, id string, name string) (string, error) {
	if !a.plugin.FunctionExists(FuncGetArtistMBID) {
		return "", agents.ErrNotFound
	}

	input := artistMBIDInput{ID: id, Name: name}
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return "", err
	}

	exit, output, err := a.plugin.Call(FuncGetArtistMBID, inputBytes)
	if err != nil {
		log.Debug(ctx, "Plugin call failed", "plugin", a.name, "function", FuncGetArtistMBID, err)
		return "", agents.ErrNotFound
	}
	if exit != 0 {
		return "", agents.ErrNotFound
	}

	var result artistMBIDOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return "", err
	}

	if result.MBID == "" {
		return "", agents.ErrNotFound
	}

	return result.MBID, nil
}

// GetArtistURL retrieves the external URL for an artist
func (a *MetadataAgent) GetArtistURL(ctx context.Context, id, name, mbid string) (string, error) {
	if !a.plugin.FunctionExists(FuncGetArtistURL) {
		return "", agents.ErrNotFound
	}

	input := artistInput{ID: id, Name: name, MBID: mbid}
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return "", err
	}

	exit, output, err := a.plugin.Call(FuncGetArtistURL, inputBytes)
	if err != nil {
		log.Debug(ctx, "Plugin call failed", "plugin", a.name, "function", FuncGetArtistURL, err)
		return "", agents.ErrNotFound
	}
	if exit != 0 {
		return "", agents.ErrNotFound
	}

	var result artistURLOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return "", err
	}

	if result.URL == "" {
		return "", agents.ErrNotFound
	}

	return result.URL, nil
}

// GetArtistBiography retrieves the biography for an artist
func (a *MetadataAgent) GetArtistBiography(ctx context.Context, id, name, mbid string) (string, error) {
	if !a.plugin.FunctionExists(FuncGetArtistBiography) {
		return "", agents.ErrNotFound
	}

	input := artistInput{ID: id, Name: name, MBID: mbid}
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return "", err
	}

	exit, output, err := a.plugin.Call(FuncGetArtistBiography, inputBytes)
	if err != nil {
		log.Debug(ctx, "Plugin call failed", "plugin", a.name, "function", FuncGetArtistBiography, err)
		return "", agents.ErrNotFound
	}
	if exit != 0 {
		return "", agents.ErrNotFound
	}

	var result artistBiographyOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return "", err
	}

	if result.Biography == "" {
		return "", agents.ErrNotFound
	}

	return result.Biography, nil
}

// GetSimilarArtists retrieves similar artists
func (a *MetadataAgent) GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]agents.Artist, error) {
	if !a.plugin.FunctionExists(FuncGetSimilarArtists) {
		return nil, agents.ErrNotFound
	}

	input := similarArtistsInput{ID: id, Name: name, MBID: mbid, Limit: limit}
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	exit, output, err := a.plugin.Call(FuncGetSimilarArtists, inputBytes)
	if err != nil {
		log.Debug(ctx, "Plugin call failed", "plugin", a.name, "function", FuncGetSimilarArtists, err)
		return nil, agents.ErrNotFound
	}
	if exit != 0 {
		return nil, agents.ErrNotFound
	}

	var result similarArtistsOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
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
	if !a.plugin.FunctionExists(FuncGetArtistImages) {
		return nil, agents.ErrNotFound
	}

	input := artistInput{ID: id, Name: name, MBID: mbid}
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	exit, output, err := a.plugin.Call(FuncGetArtistImages, inputBytes)
	if err != nil {
		log.Debug(ctx, "Plugin call failed", "plugin", a.name, "function", FuncGetArtistImages, err)
		return nil, agents.ErrNotFound
	}
	if exit != 0 {
		return nil, agents.ErrNotFound
	}

	var result artistImagesOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
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
	if !a.plugin.FunctionExists(FuncGetArtistTopSongs) {
		return nil, agents.ErrNotFound
	}

	input := topSongsInput{ID: id, Name: artistName, MBID: mbid, Count: count}
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	exit, output, err := a.plugin.Call(FuncGetArtistTopSongs, inputBytes)
	if err != nil {
		log.Debug(ctx, "Plugin call failed", "plugin", a.name, "function", FuncGetArtistTopSongs, err)
		return nil, agents.ErrNotFound
	}
	if exit != 0 {
		return nil, agents.ErrNotFound
	}

	var result topSongsOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
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
	if !a.plugin.FunctionExists(FuncGetAlbumInfo) {
		return nil, agents.ErrNotFound
	}

	input := albumInput{Name: name, Artist: artist, MBID: mbid}
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	exit, output, err := a.plugin.Call(FuncGetAlbumInfo, inputBytes)
	if err != nil {
		log.Debug(ctx, "Plugin call failed", "plugin", a.name, "function", FuncGetAlbumInfo, err)
		return nil, agents.ErrNotFound
	}
	if exit != 0 {
		return nil, agents.ErrNotFound
	}

	var result albumInfoOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
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
	if !a.plugin.FunctionExists(FuncGetAlbumImages) {
		return nil, agents.ErrNotFound
	}

	input := albumInput{Name: name, Artist: artist, MBID: mbid}
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	exit, output, err := a.plugin.Call(FuncGetAlbumImages, inputBytes)
	if err != nil {
		log.Debug(ctx, "Plugin call failed", "plugin", a.name, "function", FuncGetAlbumImages, err)
		return nil, agents.ErrNotFound
	}
	if exit != 0 {
		return nil, agents.ErrNotFound
	}

	var result albumImagesOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
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

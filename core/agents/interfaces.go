package agents

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/model"
)

type Constructor func(ds model.DataStore) Interface

type Interface interface {
	AgentName() string
}

// AlbumInfo contains album metadata (no images)
type AlbumInfo struct {
	Name        string
	MBID        string
	Description string
	URL         string
}

type Artist struct {
	ID   string
	Name string
	MBID string
}

type ExternalImage struct {
	URL  string
	Size int
}

type Song struct {
	ID         string
	Name       string
	MBID       string
	Artist     string
	ArtistMBID string
	Album      string
	AlbumMBID  string
	Duration   uint32 // Duration in milliseconds, 0 means unknown
}

var (
	ErrNotFound = errors.New("not found")
)

// AlbumInfoRetriever provides album info (no images)
type AlbumInfoRetriever interface {
	GetAlbumInfo(ctx context.Context, name, artist, mbid string) (*AlbumInfo, error)
}

// AlbumImageRetriever provides album images
type AlbumImageRetriever interface {
	GetAlbumImages(ctx context.Context, name, artist, mbid string) ([]ExternalImage, error)
}

type ArtistMBIDRetriever interface {
	GetArtistMBID(ctx context.Context, id string, name string) (string, error)
}

type ArtistURLRetriever interface {
	GetArtistURL(ctx context.Context, id, name, mbid string) (string, error)
}

type ArtistBiographyRetriever interface {
	GetArtistBiography(ctx context.Context, id, name, mbid string) (string, error)
}

type ArtistSimilarRetriever interface {
	GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]Artist, error)
}

type ArtistImageRetriever interface {
	GetArtistImages(ctx context.Context, id, name, mbid string) ([]ExternalImage, error)
}

type ArtistTopSongsRetriever interface {
	GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]Song, error)
}

// SimilarSongsByTrackRetriever provides similar songs based on a specific track
type SimilarSongsByTrackRetriever interface {
	// GetSimilarSongsByTrack returns songs similar to the given track.
	// Parameters:
	//   - id: local mediafile ID
	//   - name: track title
	//   - artist: artist name
	//   - mbid: MusicBrainz recording ID (may be empty)
	//   - count: maximum number of results
	GetSimilarSongsByTrack(ctx context.Context, id, name, artist, mbid string, count int) ([]Song, error)
}

// SimilarSongsByAlbumRetriever provides similar songs based on an album
type SimilarSongsByAlbumRetriever interface {
	// GetSimilarSongsByAlbum returns songs similar to tracks on the given album.
	// Parameters:
	//   - id: local album ID
	//   - name: album name
	//   - artist: album artist name
	//   - mbid: MusicBrainz release ID (may be empty)
	//   - count: maximum number of results
	GetSimilarSongsByAlbum(ctx context.Context, id, name, artist, mbid string, count int) ([]Song, error)
}

// SimilarSongsByArtistRetriever provides similar songs based on an artist
type SimilarSongsByArtistRetriever interface {
	// GetSimilarSongsByArtist returns songs similar to the artist's catalog.
	// Parameters:
	//   - id: local artist ID
	//   - name: artist name
	//   - mbid: MusicBrainz artist ID (may be empty)
	//   - count: maximum number of results
	GetSimilarSongsByArtist(ctx context.Context, id, name, mbid string, count int) ([]Song, error)
}

var Map map[string]Constructor

func Register(name string, init Constructor) {
	if Map == nil {
		Map = make(map[string]Constructor)
	}
	Map[name] = init
}

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
	Name string
	MBID string
}

type ExternalImage struct {
	URL  string
	Size int
}

type Song struct {
	Name string
	MBID string
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

var Map map[string]Constructor

func Register(name string, init Constructor) {
	if Map == nil {
		Map = make(map[string]Constructor)
	}
	Map[name] = init
}

package agents

import (
	"context"
	"errors"
)

type Constructor func(ctx context.Context) Interface

type Interface interface {
	AgentName() string
}

type Artist struct {
	Name string
	MBID string
}

type ArtistImage struct {
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

type ArtistMBIDRetriever interface {
	GetMBID(id string, name string) (string, error)
}

type ArtistURLRetriever interface {
	GetURL(id, name, mbid string) (string, error)
}

type ArtistBiographyRetriever interface {
	GetBiography(id, name, mbid string) (string, error)
}

type ArtistSimilarRetriever interface {
	GetSimilar(id, name, mbid string, limit int) ([]Artist, error)
}

type ArtistImageRetriever interface {
	GetImages(id, name, mbid string) ([]ArtistImage, error)
}

type ArtistTopSongsRetriever interface {
	GetTopSongs(id, artistName, mbid string, count int) ([]Song, error)
}

var Map map[string]Constructor

func Register(name string, init Constructor) {
	if Map == nil {
		Map = make(map[string]Constructor)
	}
	Map[name] = init
}

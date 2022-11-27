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
	GetMBID(ctx context.Context, id string, name string) (string, error)
}

type ArtistURLRetriever interface {
	GetURL(ctx context.Context, id, name, mbid string) (string, error)
}

type ArtistBiographyRetriever interface {
	GetBiography(ctx context.Context, id, name, mbid string) (string, error)
}

type ArtistSimilarRetriever interface {
	GetSimilar(ctx context.Context, id, name, mbid string, limit int) ([]Artist, error)
}

type ArtistImageRetriever interface {
	GetImages(ctx context.Context, id, name, mbid string) ([]ArtistImage, error)
}

type ArtistTopSongsRetriever interface {
	GetTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]Song, error)
}

var Map map[string]Constructor

func Register(name string, init Constructor) {
	if Map == nil {
		Map = make(map[string]Constructor)
	}
	Map[name] = init
}

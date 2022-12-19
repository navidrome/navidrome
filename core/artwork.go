package core

import (
	"context"
	_ "image/gif"
	_ "image/png"
	"io"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/resources"
	_ "golang.org/x/image/webp"
)

type Artwork interface {
	Get(ctx context.Context, id string, size int) (io.ReadCloser, error)
}

func NewArtwork(ds model.DataStore) Artwork {
	return &artwork{ds: ds}
}

type artwork struct {
	ds model.DataStore
}

func (a *artwork) Get(ctx context.Context, id string, size int) (io.ReadCloser, error) {
	return resources.FS().Open(consts.PlaceholderAlbumArt)
}

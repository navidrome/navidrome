package core

import (
	"context"
	"fmt"
	_ "image/gif"
	_ "image/png"
	"io"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/resources"
	"github.com/navidrome/navidrome/utils/cache"
	_ "golang.org/x/image/webp"
)

type Artwork interface {
	Get(ctx context.Context, id string, size int) (io.ReadCloser, error)
}

type ArtworkCache cache.FileCache

func NewArtwork(ds model.DataStore, cache ArtworkCache) Artwork {
	return &artwork{ds: ds, cache: cache}
}

type artwork struct {
	ds    model.DataStore
	cache cache.FileCache
}

func (a *artwork) Get(ctx context.Context, id string, size int) (io.ReadCloser, error) {
	return resources.FS().Open(consts.PlaceholderAlbumArt)
}

type imageInfo struct {
	a          *artwork
	id         string
	path       string
	size       int
	lastUpdate time.Time
}

func (ci *imageInfo) Key() string {
	return fmt.Sprintf("%s.%d.%s.%d", ci.path, ci.size, ci.lastUpdate.Format(time.RFC3339Nano), conf.Server.CoverJpegQuality)
}

var (
	onceImageCache     sync.Once
	instanceImageCache ArtworkCache
)

func GetImageCache() ArtworkCache {
	onceImageCache.Do(func() {
		instanceImageCache = cache.NewFileCache("Image", conf.Server.ImageCacheSize, consts.ImageCacheDir, consts.DefaultImageCacheMaxItems,
			func(ctx context.Context, arg cache.Item) (io.Reader, error) {
				return nil, nil
			})
	})
	return instanceImageCache
}

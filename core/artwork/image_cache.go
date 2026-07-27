package artwork

import (
	"context"
	"io"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/utils/cache"
	"github.com/navidrome/navidrome/utils/singleton"
)

// artworkReader is the cache.Item the image cache loader dispatches on: Reader
// produces the (possibly resized) bytes to store under Key.
type artworkReader interface {
	cache.Item
	Reader(ctx context.Context) (io.ReadCloser, error)
}

type imageCache struct {
	cache.FileCache
}

func GetImageCache() cache.FileCache {
	return singleton.GetInstance(func() *imageCache {
		return &imageCache{
			FileCache: cache.NewFileCache("Image", conf.Server.ImageCacheSize, consts.ImageCacheDir, consts.DefaultImageCacheMaxItems,
				func(ctx context.Context, arg cache.Item) (io.Reader, error) {
					return arg.(artworkReader).Reader(ctx)
				}),
		}
	})
}

package artwork

import (
	"bytes"
	"context"
	"io"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/ffmpeg"
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

// resizedItem is an artworkReader that resizes bytes opened by open() and caches the
// result under a hash-derived key.
type resizedItem struct {
	hash   string
	size   int
	square bool
	ffmpeg ffmpeg.FFmpeg
	open   func() (io.ReadCloser, error)
}

// Key is the ETag namespaced for the cache, so the validator a client holds and the entry it
// validates can never drift apart.
func (r *resizedItem) Key() string {
	return "h-" + representationTag(r.hash, r.size, r.square)
}

func (r *resizedItem) Reader(ctx context.Context) (io.ReadCloser, error) {
	orig, err := r.open()
	if err != nil {
		return nil, err
	}
	// An open() that reports "no image" as a nil reader would otherwise panic on the Close below.
	if orig == nil {
		return nil, ErrUnavailable
	}
	defer orig.Close()
	data, err := readCapped(orig)
	if err != nil {
		return nil, err
	}
	resized, _, err := resizeImageData(ctx, r.ffmpeg, data, r.size, r.square)
	if err != nil || resized == nil {
		// Resize failed or image already within bounds: serve the original bytes.
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	if rc, ok := resized.(io.ReadCloser); ok {
		return rc, nil
	}
	return io.NopCloser(resized), nil
}

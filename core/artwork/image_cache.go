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
	// deferAnimated makes Reader return a *deferredAnimation for animated GIFs instead of converting inline.
	deferAnimated bool
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
	if r.deferAnimated && isAnimatedGIF(data) && r.ffmpeg.IsAvailable() {
		static, _, err := resizeStaticImage(data, r.size, r.square)
		return nil, &deferredAnimation{data: data, standIn: orOriginal(data, static, err)}
	}
	resized, _, err := resizeImageData(ctx, r.ffmpeg, data, r.size, r.square)
	return io.NopCloser(orOriginal(data, resized, err)), nil
}

// orOriginal serves the original bytes when the resize failed or the image was already within bounds.
func orOriginal(data []byte, resized io.Reader, err error) io.Reader {
	if err != nil || resized == nil {
		return bytes.NewReader(data)
	}
	return resized
}

// deferredAnimation is returned as an error so the cache stores nothing: it carries the GIF bytes
// to convert in the background and a static stand-in to serve meanwhile.
type deferredAnimation struct {
	data    []byte
	standIn io.Reader
}

func (*deferredAnimation) Error() string { return "animated image conversion deferred" }

// warmCache stores item in c, returning once the entry is written.
func warmCache(ctx context.Context, c cache.FileCache, item cache.Item) error {
	stream, err := c.Get(ctx, item)
	if err != nil {
		return err
	}
	defer stream.Close()
	_, err = io.Copy(io.Discard, stream)
	return err
}

// storedItem caches bytes produced outside the cache.
type storedItem struct {
	key  string
	data []byte
}

func (i *storedItem) Key() string { return i.key }

func (i *storedItem) Reader(context.Context) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(i.data)), nil
}

package artwork

import (
	"context"
	"errors"
	"fmt"
	_ "image/gif"
	"io"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/cache"
	"github.com/navidrome/navidrome/utils/singleton"
	_ "golang.org/x/image/webp"
)

type Artwork interface {
	Get(ctx context.Context, id string, size int) (io.ReadCloser, error)
}

func NewArtwork(ds model.DataStore, cache cache.FileCache, ffmpeg ffmpeg.FFmpeg) Artwork {
	return &artwork{ds: ds, cache: cache, ffmpeg: ffmpeg}
}

type artwork struct {
	ds     model.DataStore
	cache  cache.FileCache
	ffmpeg ffmpeg.FFmpeg
}

type artworkReader interface {
	cache.Item
	LastUpdated() time.Time
	Reader(ctx context.Context) (io.ReadCloser, string, error)
}

func (a *artwork) Get(ctx context.Context, id string, size int) (reader io.ReadCloser, err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var artID model.ArtworkID
	if id != "" {
		artID, err = model.ParseArtworkID(id)
		if err != nil {
			return nil, errors.New("invalid ID")
		}
	}

	var artReader artworkReader
	switch artID.Kind {
	case model.KindAlbumArtwork:
		artReader, err = newAlbumArtworkReader(ctx, a, artID)
	case model.KindMediaFileArtwork:
		artReader, err = newMediafileArtworkReader(ctx, a, artID)
	default:
		artReader, err = newEmptyIDReader(ctx, artID)
	}
	if err != nil {
		return nil, err
	}
	if size > 0 {
		artReader = resizedFromOriginal(artReader, artID, size)
	}

	r, err := a.cache.Get(ctx, artReader)
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Error(ctx, "Error accessing image cache", "id", id, "size", size, err)
	}
	return r, err
}

type cacheItem struct {
	artID      model.ArtworkID
	size       int
	lastUpdate time.Time
}

func (i *cacheItem) Key() string {
	return fmt.Sprintf("%s.%d.%d.%d", i.artID.ID, i.lastUpdate.UnixMilli(), i.size, conf.Server.CoverJpegQuality)
}

type imageCache struct {
	cache.FileCache
}

func GetImageCache() cache.FileCache {
	return singleton.GetInstance(func() *imageCache {
		return &imageCache{
			FileCache: cache.NewFileCache("Image", conf.Server.ImageCacheSize, consts.ImageCacheDir, consts.DefaultImageCacheMaxItems,
				func(ctx context.Context, arg cache.Item) (io.Reader, error) {
					r, _, err := arg.(artworkReader).Reader(ctx)
					return r, err
				}),
		}
	})
}

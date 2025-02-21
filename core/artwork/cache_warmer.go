package artwork

import (
	"context"
	"fmt"
	"io"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/cache"
	"github.com/navidrome/navidrome/utils/pl"
)

type CacheWarmer interface {
	PreCache(artID model.ArtworkID)
}

// NewCacheWarmer creates a new CacheWarmer instance. The CacheWarmer will pre-cache Artwork images in the background
// to speed up the response time when the image is requested by the UI. The cache is pre-populated with the original
// image size, as well as the size defined in the UICoverArtSize constant.
func NewCacheWarmer(artwork Artwork, cache cache.FileCache) CacheWarmer {
	// If image cache is disabled, return a NOOP implementation
	if conf.Server.ImageCacheSize == "0" || !conf.Server.EnableArtworkPrecache {
		return &noopCacheWarmer{}
	}

	a := &cacheWarmer{
		artwork:    artwork,
		cache:      cache,
		buffer:     make(map[model.ArtworkID]struct{}),
		wakeSignal: make(chan struct{}, 1),
	}

	// Create a context with a fake admin user, to be able to pre-cache Playlist CoverArts
	ctx := request.WithUser(context.TODO(), model.User{IsAdmin: true})
	go a.run(ctx)
	return a
}

type cacheWarmer struct {
	artwork    Artwork
	buffer     map[model.ArtworkID]struct{}
	mutex      sync.Mutex
	cache      cache.FileCache
	wakeSignal chan struct{}
}

func (a *cacheWarmer) PreCache(artID model.ArtworkID) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.buffer[artID] = struct{}{}
	a.sendWakeSignal()
}

func (a *cacheWarmer) sendWakeSignal() {
	// Don't block if the previous signal was not read yet
	select {
	case a.wakeSignal <- struct{}{}:
	default:
	}
}

func (a *cacheWarmer) run(ctx context.Context) {
	for {
		a.waitSignal(ctx, 10*time.Second)
		if ctx.Err() != nil {
			break
		}

		// If cache not available, keep waiting
		if !a.cache.Available(ctx) {
			if len(a.buffer) > 0 {
				log.Trace(ctx, "Cache not available, buffering precache request", "bufferLen", len(a.buffer))
			}
			continue
		}

		a.mutex.Lock()

		// If there's nothing to send, keep waiting
		if len(a.buffer) == 0 {
			a.mutex.Unlock()
			continue
		}

		batch := slices.Collect(maps.Keys(a.buffer))
		a.buffer = make(map[model.ArtworkID]struct{})
		a.mutex.Unlock()

		a.processBatch(ctx, batch)
	}
}

func (a *cacheWarmer) waitSignal(ctx context.Context, timeout time.Duration) {
	select {
	case <-time.After(timeout):
	case <-a.wakeSignal:
	case <-ctx.Done():
	}
}

func (a *cacheWarmer) processBatch(ctx context.Context, batch []model.ArtworkID) {
	log.Trace(ctx, "PreCaching a new batch of artwork", "batchSize", len(batch))
	input := pl.FromSlice(ctx, batch)
	errs := pl.Sink(ctx, 2, input, a.doCacheImage)
	for err := range errs {
		log.Debug(ctx, "Error warming cache", err)
	}
}

func (a *cacheWarmer) doCacheImage(ctx context.Context, id model.ArtworkID) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	r, _, err := a.artwork.Get(ctx, id, consts.UICoverArtSize, true)
	if err != nil {
		return fmt.Errorf("caching id='%s': %w", id, err)
	}
	defer r.Close()
	_, err = io.Copy(io.Discard, r)
	if err != nil {
		return err
	}
	return nil
}

func NoopCacheWarmer() CacheWarmer {
	return &noopCacheWarmer{}
}

type noopCacheWarmer struct{}

func (a *noopCacheWarmer) PreCache(model.ArtworkID) {}

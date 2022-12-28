package artwork

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/pl"
)

type CacheWarmer interface {
	PreCache(artID model.ArtworkID)
}

func NewCacheWarmer(artwork Artwork) CacheWarmer {
	// If image cache is disabled, return a NOOP implementation
	if conf.Server.ImageCacheSize == "0" {
		return &noopCacheWarmer{}
	}

	a := &cacheWarmer{
		artwork:    artwork,
		wakeSignal: make(chan struct{}, 1),
	}

	// Create a context with a fake admin user, to be able to pre-cache Playlist CoverArts
	ctx := request.WithUser(context.TODO(), model.User{IsAdmin: true})
	go a.run(ctx)
	return a
}

type cacheWarmer struct {
	artwork    Artwork
	buffer     []string
	mutex      sync.Mutex
	wakeSignal chan struct{}
}

func (a *cacheWarmer) PreCache(artID model.ArtworkID) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.buffer = append(a.buffer, artID.String())
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
		time.AfterFunc(5*time.Second, func() {
			a.sendWakeSignal()
		})
		<-a.wakeSignal

		a.mutex.Lock()
		var batch []string
		if len(a.buffer) > 0 {
			batch = a.buffer
			a.buffer = nil
		}
		a.mutex.Unlock()

		if len(batch) > 0 {
			a.processBatch(ctx, batch)
		}
	}
}

func (a *cacheWarmer) processBatch(ctx context.Context, batch []string) {
	log.Trace(ctx, "PreCaching a new batch of artwork", "batchSize", len(batch))
	input := pl.FromSlice(ctx, batch)
	errs := pl.Sink(ctx, 2, input, a.doCacheImage)
	for err := range errs {
		log.Warn(ctx, "Error warming cache", err)
	}
}

func (a *cacheWarmer) doCacheImage(ctx context.Context, id string) error {
	r, _, err := a.artwork.Get(ctx, id, 0)
	if err != nil {
		return fmt.Errorf("error cacheing id='%s': %w", id, err)
	}
	defer r.Close()
	_, err = io.Copy(io.Discard, r)
	if err != nil {
		return err
	}
	return nil
}

type noopCacheWarmer struct{}

func (a *noopCacheWarmer) PreCache(id model.ArtworkID) {}

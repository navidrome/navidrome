package core

import (
	"context"
	"io"
	"io/ioutil"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/pool"
)

type CacheWarmer interface {
	AddAlbum(ctx context.Context, albumID string)
	Flush(ctx context.Context)
}

func NewCacheWarmer(artwork Artwork, artworkCache ArtworkCache) CacheWarmer {
	w := &warmer{
		artwork:      artwork,
		artworkCache: artworkCache,
		albums:       map[string]struct{}{},
	}
	p, err := pool.NewPool("artwork", 3, w.execute)
	if err != nil {
		log.Error(context.Background(), "Error creating pool for Album Artwork Cache Warmer", err)
	} else {
		w.pool = p
	}

	return w
}

type warmer struct {
	pool         *pool.Pool
	artwork      Artwork
	artworkCache ArtworkCache
	albums       map[string]struct{}
}

func (w *warmer) AddAlbum(ctx context.Context, albumID string) {
	if albumID == "" {
		return
	}
	w.albums[albumID] = struct{}{}
}

func (w *warmer) waitForCacheReady(ctx context.Context) {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		<-tick.C
		if w.artworkCache.Ready(ctx) {
			return
		}
	}
}

func (w *warmer) Flush(ctx context.Context) {
	if conf.Server.DevPreCacheAlbumArtwork {
		w.waitForCacheReady(ctx)
		if w.artworkCache.Available(ctx) {
			if w.pool == nil || len(w.albums) == 0 {
				return
			}
			log.Info(ctx, "Pre-caching album artworks", "numAlbums", len(w.albums))
			for id := range w.albums {
				w.pool.Submit(artworkItem{albumID: id})
			}
		} else {
			log.Warn(ctx, "Cache warmer is not available as ImageCache is DISABLED")
		}
	}
	w.albums = map[string]struct{}{}
}

func (w *warmer) execute(workload interface{}) {
	ctx := context.Background()
	item := workload.(artworkItem)
	log.Trace(ctx, "Pre-caching album artwork", "albumID", item.albumID)
	r, err := w.artwork.Get(ctx, item.albumID, 0)
	if err != nil {
		log.Warn("Error pre-caching artwork from album", "id", item.albumID, err)
		return
	}
	defer r.Close()
	_, _ = io.Copy(ioutil.Discard, r)
}

type artworkItem struct {
	albumID string
}

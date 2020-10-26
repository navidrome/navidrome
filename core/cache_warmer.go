package core

import (
	"context"
	"io/ioutil"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/core/pool"
	"github.com/deluan/navidrome/log"
)

type CacheWarmer interface {
	AddAlbum(ctx context.Context, albumID string)
	Flush(ctx context.Context)
}

func NewCacheWarmer(artwork Artwork) CacheWarmer {
	w := &warmer{
		artwork: artwork,
		albums:  map[string]struct{}{},
	}
	p, err := pool.NewPool("artwork", 3, &artworkItem{}, w.execute)
	if err != nil {
		log.Error(context.Background(), "Error creating pool for Album Artwork Cache Warmer", err)
	} else {
		w.pool = p
	}

	return w
}

type warmer struct {
	pool    *pool.Pool
	artwork Artwork
	albums  map[string]struct{}
}

func (w *warmer) AddAlbum(ctx context.Context, albumID string) {
	if albumID == "" {
		return
	}
	w.albums[albumID] = struct{}{}
}

func (w *warmer) Flush(ctx context.Context) {
	if conf.Server.DevPreCacheAlbumArtwork {
		if w.pool == nil || len(w.albums) == 0 {
			return
		}
		log.Info(ctx, "Pre-caching album artworks", "numAlbums", len(w.albums))
		for id := range w.albums {
			w.pool.Submit(artworkItem{albumID: id})
		}
	}
	w.albums = map[string]struct{}{}
}

func (w *warmer) execute(workload interface{}) {
	ctx := context.Background()
	item := workload.(artworkItem)
	log.Trace(ctx, "Pre-caching album artwork", "albumID", item.albumID)
	err := w.artwork.Get(ctx, item.albumID, 0, ioutil.Discard)
	if err != nil {
		log.Warn("Error pre-caching artwork from album", "id", item.albumID, err)
	}
}

type artworkItem struct {
	albumID string
}

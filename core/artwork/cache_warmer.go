package artwork

import (
	"context"
	"fmt"
	"io"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
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
		artwork: artwork,
		input:   make(chan string),
	}
	go a.run(context.TODO())
	return a
}

type cacheWarmer struct {
	artwork Artwork
	input   chan string
}

func (a *cacheWarmer) PreCache(artID model.ArtworkID) {
	a.input <- artID.String()
}

func (a *cacheWarmer) run(ctx context.Context) {
	errs := pl.Sink(ctx, 2, a.input, a.doCacheImage)
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

package scanner

import (
	"context"
	"strings"
	"time"

	"github.com/ReneKroon/ttlcache/v2"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

func newCachedPublisherRepository(ctx context.Context, repo model.PublisherRepository) model.PublisherRepository {
	r := &cachedPublisherRepo{
		PublisherRepository: repo,
		ctx:                 ctx,
	}
	publishers, err := repo.GetAll()
	if err != nil {
		log.Error(ctx, "Could not load publishers from DB", err)
		return repo
	}

	r.cache = ttlcache.NewCache()
	for _, g := range publishers {
		_ = r.cache.Set(strings.ToLower(g.Name), g.ID)
	}

	return r
}

type cachedPublisherRepo struct {
	model.PublisherRepository
	cache *ttlcache.Cache
	ctx   context.Context
}

func (r *cachedPublisherRepo) Put(g *model.Publisher) error {
	id, err := r.cache.GetByLoader(strings.ToLower(g.Name), func(key string) (interface{}, time.Duration, error) {
		err := r.PublisherRepository.Put(g)
		return g.ID, 24 * time.Hour, err
	})
	g.ID = id.(string)
	return err
}

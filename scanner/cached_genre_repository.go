package scanner

import (
	"context"
	"strings"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/cache"
	"github.com/navidrome/navidrome/utils/singleton"
)

func newCachedGenreRepository(ctx context.Context, repo model.GenreRepository) model.GenreRepository {
	return singleton.GetInstance(func() *cachedGenreRepo {
		r := &cachedGenreRepo{
			GenreRepository: repo,
			ctx:             ctx,
		}
		genres, err := repo.GetAll()

		if err != nil {
			log.Error(ctx, "Could not load genres from DB", err)
			panic(err)
		}
		r.cache = cache.NewSimpleCache[string, string]()
		for _, g := range genres {
			_ = r.cache.Add(strings.ToLower(g.Name), g.ID)
		}
		return r
	})
}

type cachedGenreRepo struct {
	model.GenreRepository
	cache cache.SimpleCache[string, string]
	ctx   context.Context
}

func (r *cachedGenreRepo) Put(g *model.Genre) error {
	id, err := r.cache.GetWithLoader(strings.ToLower(g.Name), func(key string) (string, time.Duration, error) {
		err := r.GenreRepository.Put(g)
		return g.ID, 24 * time.Hour, err
	})
	g.ID = id
	return err
}

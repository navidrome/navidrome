package engine

import (
	"context"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
)

type Ratings interface {
	SetStar(ctx context.Context, star bool, ids ...string) error
	SetRating(ctx context.Context, id string, rating int) error
}

func NewRatings(ds model.DataStore) Ratings {
	return &ratings{ds}
}

type ratings struct {
	ds model.DataStore
}

func (r ratings) SetRating(ctx context.Context, id string, rating int) error {
	exist, err := r.ds.Album(ctx).Exists(id)
	if err != nil {
		return err
	}
	if exist {
		return r.ds.Album(ctx).SetRating(rating, id)
	}
	return r.ds.MediaFile(ctx).SetRating(rating, id)
}

func (r ratings) SetStar(ctx context.Context, star bool, ids ...string) error {
	if len(ids) == 0 {
		log.Warn(ctx, "Cannot star/unstar an empty list of ids")
		return nil
	}

	return r.ds.WithTx(func(tx model.DataStore) error {
		for _, id := range ids {
			exist, err := r.ds.Album(ctx).Exists(id)
			if err != nil {
				return err
			}
			if exist {
				err = tx.Album(ctx).SetStar(star, ids...)
				if err != nil {
					return err
				}
				continue
			}
			exist, err = r.ds.Artist(ctx).Exists(id)
			if err != nil {
				return err
			}
			if exist {
				err = tx.Artist(ctx).SetStar(star, ids...)
				if err != nil {
					return err
				}
				continue
			}
			err = tx.MediaFile(ctx).SetStar(star, ids...)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

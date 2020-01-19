package engine

import (
	"context"

	"github.com/cloudsonic/sonic-server/model"
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
	// TODO
	return model.ErrNotFound
}

func (r ratings) SetStar(ctx context.Context, star bool, ids ...string) error {
	return r.ds.WithTx(func(tx model.DataStore) error {
		err := tx.MediaFile().SetStar(star, ids...)
		if err != nil {
			return err
		}
		err = tx.Album().SetStar(star, ids...)
		if err != nil {
			return err
		}
		err = tx.Artist().SetStar(star, ids...)
		return err
	})
}

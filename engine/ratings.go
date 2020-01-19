package engine

import (
	"context"

	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/itunesbridge"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/model"
	"github.com/cloudsonic/sonic-server/utils"
)

type Ratings interface {
	SetStar(ctx context.Context, star bool, ids ...string) error
	SetRating(ctx context.Context, id string, rating int) error
}

func NewRatings(itunes itunesbridge.ItunesControl, mr model.MediaFileRepository, alr model.AlbumRepository, ar model.ArtistRepository) Ratings {
	return &ratings{itunes, mr, alr, ar}
}

type ratings struct {
	itunes     itunesbridge.ItunesControl
	mfRepo     model.MediaFileRepository
	albumRepo  model.AlbumRepository
	artistRepo model.ArtistRepository
}

func (r ratings) SetRating(ctx context.Context, id string, rating int) error {
	rating = utils.MinInt(rating, 5) * 20

	isAlbum, _ := r.albumRepo.Exists(id)
	if isAlbum {
		mfs, _ := r.mfRepo.FindByAlbum(id)
		if len(mfs) > 0 {
			log.Debug(ctx, "Set Rating", "value", rating, "album", mfs[0].Album)
			if err := r.itunes.SetAlbumRating(mfs[0].ID, rating); err != nil {
				return err
			}
		}
		return nil
	}

	mf, err := r.mfRepo.Get(id)
	if err != nil {
		return err
	}
	if mf != nil {
		log.Debug(ctx, "Set Rating", "value", rating, "song", mf.Title)
		if err := r.itunes.SetTrackRating(mf.ID, rating); err != nil {
			return err
		}
		return nil
	}
	return model.ErrNotFound
}

func (r ratings) SetStar(ctx context.Context, star bool, ids ...string) error {
	if conf.Sonic.DevUseFileScanner {
		err := r.mfRepo.SetStar(star, ids...)
		if err != nil {
			return err
		}
		err = r.albumRepo.SetStar(star, ids...)
		if err != nil {
			return err
		}
		err = r.artistRepo.SetStar(star, ids...)
		return err
	}

	for _, id := range ids {
		isAlbum, _ := r.albumRepo.Exists(id)
		if isAlbum {
			mfs, _ := r.mfRepo.FindByAlbum(id)
			if len(mfs) > 0 {
				log.Debug(ctx, "Set Star", "value", star, "album", mfs[0].Album)
				if err := r.itunes.SetAlbumLoved(mfs[0].ID, star); err != nil {
					return err
				}
			}
			continue
		}

		mf, err := r.mfRepo.Get(id)
		if err != nil {
			return err
		}
		if mf != nil {
			log.Debug(ctx, "Set Star", "value", star, "song", mf.Title)
			if err := r.itunes.SetTrackLoved(mf.ID, star); err != nil {
				return err
			}
			continue
		}
		return model.ErrNotFound
	}

	return nil
}

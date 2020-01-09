package engine

import (
	"context"

	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/itunesbridge"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/utils"
)

type Ratings interface {
	SetStar(ctx context.Context, star bool, ids ...string) error
	SetRating(ctx context.Context, id string, rating int) error
}

func NewRatings(itunes itunesbridge.ItunesControl, mr domain.MediaFileRepository, alr domain.AlbumRepository, ar domain.ArtistRepository) Ratings {
	return &ratings{itunes, mr, alr, ar}
}

type ratings struct {
	itunes     itunesbridge.ItunesControl
	mfRepo     domain.MediaFileRepository
	albumRepo  domain.AlbumRepository
	artistRepo domain.ArtistRepository
}

func (r ratings) SetRating(ctx context.Context, id string, rating int) error {
	rating = utils.MinInt(rating, 5) * 20

	isAlbum, _ := r.albumRepo.Exists(id)
	if isAlbum {
		mfs, _ := r.mfRepo.FindByAlbum(id)
		if len(mfs) > 0 {
			log.Debug(ctx, "Set Rating", "value", rating, "album", mfs[0].Album)
			if err := r.itunes.SetAlbumRating(mfs[0].Id, rating); err != nil {
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
		if err := r.itunes.SetTrackRating(mf.Id, rating); err != nil {
			return err
		}
		return nil
	}
	return domain.ErrNotFound
}

func (r ratings) SetStar(ctx context.Context, star bool, ids ...string) error {
	for _, id := range ids {
		isAlbum, _ := r.albumRepo.Exists(id)
		if isAlbum {
			mfs, _ := r.mfRepo.FindByAlbum(id)
			if len(mfs) > 0 {
				log.Debug(ctx, "Set Star", "value", star, "album", mfs[0].Album)
				if err := r.itunes.SetAlbumLoved(mfs[0].Id, star); err != nil {
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
			if err := r.itunes.SetTrackLoved(mf.Id, star); err != nil {
				return err
			}
			continue
		}
		return domain.ErrNotFound
	}

	return nil
}

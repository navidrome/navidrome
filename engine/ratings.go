package engine

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/itunesbridge"
)

type Ratings interface {
	SetStar(star bool, ids ...string) error
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

func (r ratings) SetStar(star bool, ids ...string) error {
	for _, id := range ids {
		isAlbum, _ := r.albumRepo.Exists(id)
		if isAlbum {
			mfs, _ := r.mfRepo.FindByAlbum(id)
			if len(mfs) > 0 {
				beego.Debug("SetStar:", star, "Album:", mfs[0].Album)
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
			beego.Debug("SetStar:", star, "Song:", mf.Title)
			if err := r.itunes.SetTrackLoved(mf.Id, star); err != nil {
				return err
			}
			continue
		}
		return ErrDataNotFound
	}

	return nil
}

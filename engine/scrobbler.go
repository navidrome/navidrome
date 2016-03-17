package engine

import (
	"errors"
	"fmt"
	"time"

	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/itunesbridge"
)

type Scrobbler interface {
	Register(id string, playDate time.Time, submit bool) (*domain.MediaFile, error)
}

func NewScrobbler(itunes itunesbridge.ItunesControl, mr domain.MediaFileRepository, npr NowPlayingRepository) Scrobbler {
	return scrobbler{itunes, mr, npr}
}

type scrobbler struct {
	itunes itunesbridge.ItunesControl
	mfRepo domain.MediaFileRepository
	npRepo NowPlayingRepository
}

func (s scrobbler) Register(id string, playDate time.Time, submit bool) (*domain.MediaFile, error) {
	mf, err := s.mfRepo.Get(id)
	if err != nil {
		return nil, err
	}

	if mf == nil {
		return nil, errors.New(fmt.Sprintf(`Id "%s" not found`, id))
	}

	if submit {
		if err := s.itunes.MarkAsPlayed(id, playDate); err != nil {
			return nil, err
		}
	}
	return mf, nil
}

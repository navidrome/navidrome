package engine

import (
	"errors"
	"fmt"
	"time"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/itunesbridge"
)

type Scrobbler interface {
	Register(playerId int, trackId string, playDate time.Time) (*domain.MediaFile, error)
	NowPlaying(playerId int, playerName, trackId, username string) (*domain.MediaFile, error)
}

func NewScrobbler(itunes itunesbridge.ItunesControl, mr domain.MediaFileRepository, npr NowPlayingRepository) Scrobbler {
	return &scrobbler{itunes, mr, npr}
}

type scrobbler struct {
	itunes itunesbridge.ItunesControl
	mfRepo domain.MediaFileRepository
	npRepo NowPlayingRepository
}

func (s *scrobbler) Register(playerId int, trackId string, playDate time.Time) (*domain.MediaFile, error) {
	for {
		np, err := s.npRepo.Dequeue(playerId)
		if err != nil || np == nil || np.TrackId == trackId {
			break
		}
		err = s.itunes.MarkAsSkipped(np.TrackId, np.Start.Add(time.Duration(1)*time.Minute))
		if err != nil {
			beego.Warn("Error skipping track", np.TrackId)
		} else {
			beego.Debug("Skipped track", np.TrackId)
		}
	}

	mf, err := s.mfRepo.Get(trackId)
	if err != nil {
		return nil, err
	}

	if mf == nil {
		return nil, errors.New(fmt.Sprintf(`Id "%s" not found`, trackId))
	}

	if err := s.itunes.MarkAsPlayed(trackId, playDate); err != nil {
		return nil, err
	}
	return mf, nil
}

func (s *scrobbler) NowPlaying(playerId int, playerName, trackId, username string) (*domain.MediaFile, error) {
	mf, err := s.mfRepo.Get(trackId)
	if err != nil {
		return nil, err
	}

	if mf == nil {
		return nil, errors.New(fmt.Sprintf(`Id "%s" not found`, trackId))
	}

	return mf, s.npRepo.Enqueue(playerId, playerName, trackId, username)
}

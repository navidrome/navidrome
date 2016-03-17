package engine

import (
	"errors"
	"fmt"
	"time"

	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/itunesbridge"
)

type Scrobbler interface {
	Register(playerId int, trackId string, playDate time.Time) (*domain.MediaFile, error)
	NowPlaying(playerId int, trackId, username string, playerName string) (*domain.MediaFile, error)
	DetectSkipped(playerId int, trackId string, submission bool) (bool, error)
}

func NewScrobbler(itunes itunesbridge.ItunesControl, mr domain.MediaFileRepository, npr NowPlayingRepository) Scrobbler {
	return &scrobbler{itunes, mr, npr}
}

type scrobbler struct {
	itunes itunesbridge.ItunesControl
	mfRepo domain.MediaFileRepository
	npRepo NowPlayingRepository
}

func (s *scrobbler) DetectSkipped(playerId int, trackId string, submission bool) (bool, error) {
	np, err := s.npRepo.Clear(playerId)
	if err != nil {
		return false, err
	}

	if np == nil {
		return false, nil
	}

	if (submission && np.TrackId != trackId) || (!submission) {
		return true, s.itunes.MarkAsSkipped(np.TrackId, time.Now())
	}
	return false, nil
}

func (s *scrobbler) Register(playerId int, id string, playDate time.Time) (*domain.MediaFile, error) {
	mf, err := s.mfRepo.Get(id)
	if err != nil {
		return nil, err
	}

	if mf == nil {
		return nil, errors.New(fmt.Sprintf(`Id "%s" not found`, id))
	}

	if err := s.itunes.MarkAsPlayed(id, playDate); err != nil {
		return nil, err
	}
	return mf, nil
}

func (s *scrobbler) NowPlaying(playerId int, trackId, username string, playerName string) (*domain.MediaFile, error) {
	mf, err := s.mfRepo.Get(trackId)
	if err != nil {
		return nil, err
	}

	if mf == nil {
		return nil, errors.New(fmt.Sprintf(`Id "%s" not found`, trackId))
	}

	return mf, s.npRepo.Set(trackId, username, playerId, playerName)
}

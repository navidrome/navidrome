package engine

import (
	"errors"
	"fmt"
	"time"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/itunesbridge"
)

const (
	minSkipped = time.Duration(3) * time.Second
	maxSkipped = time.Duration(20) * time.Second
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

func (s *scrobbler) detectSkipped(playerId int, trackId string) {
	for {
		size, _ := s.npRepo.Count(playerId)
		np, err := s.npRepo.Tail(playerId)
		if err != nil || np == nil || (size == 1 && np.TrackId != trackId) {
			break
		}

		s.npRepo.Dequeue(playerId)
		if np.TrackId == trackId {
			break
		}
		err = s.itunes.MarkAsSkipped(np.TrackId, np.Start.Add(time.Duration(1)*time.Minute))
		if err != nil {
			beego.Warn("Error skipping track", np.TrackId)
		} else {
			beego.Debug("Skipped track", np.TrackId)
		}
	}
}

func (s *scrobbler) Register(playerId int, trackId string, playTime time.Time) (*domain.MediaFile, error) {
	s.detectSkipped(playerId, trackId)

	mf, err := s.mfRepo.Get(trackId)
	if err != nil {
		return nil, err
	}

	if mf == nil {
		return nil, errors.New(fmt.Sprintf(`Id "%s" not found`, trackId))
	}

	if err := s.itunes.MarkAsPlayed(trackId, playTime); err != nil {
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

	info := &NowPlayingInfo{TrackId: trackId, Username: username, Start: time.Now(), PlayerId: playerId, PlayerName: playerName}
	return mf, s.npRepo.Enqueue(info)
}

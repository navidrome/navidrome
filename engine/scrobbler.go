package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cloudsonic/sonic-server/itunesbridge"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/model"
)

const (
	minSkipped = 3 * time.Second
	maxSkipped = 20 * time.Second
)

type Scrobbler interface {
	Register(ctx context.Context, playerId int, trackId string, playDate time.Time) (*model.MediaFile, error)
	NowPlaying(ctx context.Context, playerId int, playerName, trackId, username string) (*model.MediaFile, error)
}

func NewScrobbler(itunes itunesbridge.ItunesControl, mr model.MediaFileRepository, npr NowPlayingRepository) Scrobbler {
	return &scrobbler{itunes, mr, npr}
}

type scrobbler struct {
	itunes itunesbridge.ItunesControl
	mfRepo model.MediaFileRepository
	npRepo NowPlayingRepository
}

func (s *scrobbler) detectSkipped(ctx context.Context, playerId int, trackId string) {
	size, _ := s.npRepo.Count(playerId)
	switch size {
	case 0:
		return
	case 1:
		np, _ := s.npRepo.Tail(playerId)
		if np.TrackID != trackId {
			return
		}
		s.npRepo.Dequeue(playerId)
	default:
		prev, _ := s.npRepo.Dequeue(playerId)
		for {
			if prev.TrackID == trackId {
				break
			}
			np, err := s.npRepo.Dequeue(playerId)
			if np == nil || err != nil {
				break
			}
			diff := np.Start.Sub(prev.Start)
			if diff < minSkipped || diff > maxSkipped {
				log.Debug(ctx, fmt.Sprintf("-- Playtime for track %s was %v. Not skipping.", prev.TrackID, diff))
				prev = np
				continue
			}
			err = s.itunes.MarkAsSkipped(prev.TrackID, prev.Start.Add(1*time.Minute))
			if err != nil {
				log.Warn(ctx, "Error skipping track", "id", prev.TrackID)
			} else {
				log.Debug(ctx, "-- Skipped track "+prev.TrackID)
			}
		}
	}
}

func (s *scrobbler) Register(ctx context.Context, playerId int, trackId string, playTime time.Time) (*model.MediaFile, error) {
	s.detectSkipped(ctx, playerId, trackId)

	mf, err := s.mfRepo.Get(trackId)
	if err != nil {
		return nil, err
	}

	if mf == nil {
		return nil, errors.New(fmt.Sprintf(`ID "%s" not found`, trackId))
	}

	if err := s.itunes.MarkAsPlayed(trackId, playTime); err != nil {
		return nil, err
	}
	return mf, nil
}

func (s *scrobbler) NowPlaying(ctx context.Context, playerId int, playerName, trackId, username string) (*model.MediaFile, error) {
	mf, err := s.mfRepo.Get(trackId)
	if err != nil {
		return nil, err
	}

	if mf == nil {
		return nil, errors.New(fmt.Sprintf(`ID "%s" not found`, trackId))
	}

	info := &NowPlayingInfo{TrackID: trackId, Username: username, Start: time.Now(), PlayerId: playerId, PlayerName: playerName}
	return mf, s.npRepo.Enqueue(info)
}

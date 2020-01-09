package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/itunesbridge"
	"github.com/cloudsonic/sonic-server/log"
)

const (
	minSkipped = 3 * time.Second
	maxSkipped = 20 * time.Second
)

type Scrobbler interface {
	Register(ctx context.Context, playerId int, trackId string, playDate time.Time) (*domain.MediaFile, error)
	NowPlaying(ctx context.Context, playerId int, playerName, trackId, username string) (*domain.MediaFile, error)
}

func NewScrobbler(itunes itunesbridge.ItunesControl, mr domain.MediaFileRepository, npr NowPlayingRepository) Scrobbler {
	return &scrobbler{itunes, mr, npr}
}

type scrobbler struct {
	itunes itunesbridge.ItunesControl
	mfRepo domain.MediaFileRepository
	npRepo NowPlayingRepository
}

func (s *scrobbler) detectSkipped(ctx context.Context, playerId int, trackId string) {
	size, _ := s.npRepo.Count(playerId)
	switch size {
	case 0:
		return
	case 1:
		np, _ := s.npRepo.Tail(playerId)
		if np.TrackId != trackId {
			return
		}
		s.npRepo.Dequeue(playerId)
	default:
		prev, _ := s.npRepo.Dequeue(playerId)
		for {
			if prev.TrackId == trackId {
				break
			}
			np, err := s.npRepo.Dequeue(playerId)
			if np == nil || err != nil {
				break
			}
			diff := np.Start.Sub(prev.Start)
			if diff < minSkipped || diff > maxSkipped {
				log.Debug(ctx, fmt.Sprintf("-- Playtime for track %s was %v. Not skipping.", prev.TrackId, diff))
				prev = np
				continue
			}
			err = s.itunes.MarkAsSkipped(prev.TrackId, prev.Start.Add(1*time.Minute))
			if err != nil {
				log.Warn(ctx, "Error skipping track", "id", prev.TrackId)
			} else {
				log.Debug(ctx, "-- Skipped track "+prev.TrackId)
			}
		}
	}
}

func (s *scrobbler) Register(ctx context.Context, playerId int, trackId string, playTime time.Time) (*domain.MediaFile, error) {
	s.detectSkipped(ctx, playerId, trackId)

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

func (s *scrobbler) NowPlaying(ctx context.Context, playerId int, playerName, trackId, username string) (*domain.MediaFile, error) {
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

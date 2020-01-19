package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func NewScrobbler(ds model.DataStore, npr NowPlayingRepository) Scrobbler {
	return &scrobbler{ds: ds, npRepo: npr}
}

type scrobbler struct {
	ds     model.DataStore
	npRepo NowPlayingRepository
}

func (s *scrobbler) Register(ctx context.Context, playerId int, trackId string, playTime time.Time) (*model.MediaFile, error) {
	// TODO Add transaction
	mf, err := s.ds.MediaFile().Get(trackId)
	if err != nil {
		return nil, err
	}
	err = s.ds.MediaFile().MarkAsPlayed(trackId, playTime)
	if err != nil {
		return nil, err
	}
	err = s.ds.Album().MarkAsPlayed(mf.AlbumID, playTime)
	return mf, err
}

func (s *scrobbler) NowPlaying(ctx context.Context, playerId int, playerName, trackId, username string) (*model.MediaFile, error) {
	mf, err := s.ds.MediaFile().Get(trackId)
	if err != nil {
		return nil, err
	}

	if mf == nil {
		return nil, errors.New(fmt.Sprintf(`ID "%s" not found`, trackId))
	}

	info := &NowPlayingInfo{TrackID: trackId, Username: username, Start: time.Now(), PlayerId: playerId, PlayerName: playerName}
	return mf, s.npRepo.Enqueue(info)
}

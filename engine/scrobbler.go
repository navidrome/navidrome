package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/deluan/navidrome/model"
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
	var mf *model.MediaFile
	var err error
	err = s.ds.WithTx(func(tx model.DataStore) error {
		mf, err = s.ds.MediaFile(ctx).Get(trackId)
		if err != nil {
			return err
		}
		err = s.ds.Annotation(ctx).IncPlayCount(model.MediaItemType, trackId, playTime)
		if err != nil {
			return err
		}
		err = s.ds.Annotation(ctx).IncPlayCount(model.AlbumItemType, mf.AlbumID, playTime)
		return err
	})
	return mf, err
}

// TODO Validate if NowPlaying still works after all refactorings
func (s *scrobbler) NowPlaying(ctx context.Context, playerId int, playerName, trackId, username string) (*model.MediaFile, error) {
	mf, err := s.ds.MediaFile(ctx).Get(trackId)
	if err != nil {
		return nil, err
	}

	if mf == nil {
		return nil, errors.New(fmt.Sprintf(`ID "%s" not found`, trackId))
	}

	info := &NowPlayingInfo{TrackID: trackId, Username: username, Start: time.Now(), PlayerId: playerId, PlayerName: playerName}
	return mf, s.npRepo.Enqueue(info)
}

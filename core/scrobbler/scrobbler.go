package scrobbler

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/singleton"
)

const nowPlayingExpire = 60 * time.Minute

type NowPlayingInfo struct {
	TrackID    string
	Start      time.Time
	Username   string
	PlayerId   int
	PlayerName string
}

type Scrobbler interface {
	NowPlaying(ctx context.Context, playerId int, playerName string, trackId string) error
	GetNowPlaying(ctx context.Context) ([]NowPlayingInfo, error)
	Submit(ctx context.Context, playerId int, trackId string, playTime time.Time) error
}

type scrobbler struct {
	ds model.DataStore
}

var playMap = sync.Map{}

func New(ds model.DataStore) Scrobbler {
	instance := singleton.Get(scrobbler{}, func() interface{} {
		return &scrobbler{ds: ds}
	})
	return instance.(*scrobbler)
}

func (s *scrobbler) NowPlaying(ctx context.Context, playerId int, playerName string, trackId string) error {
	username, _ := request.UsernameFrom(ctx)
	info := NowPlayingInfo{
		TrackID:    trackId,
		Start:      time.Now(),
		Username:   username,
		PlayerId:   playerId,
		PlayerName: playerName,
	}
	playMap.Store(playerId, info)
	return nil
}

func (s *scrobbler) GetNowPlaying(ctx context.Context) ([]NowPlayingInfo, error) {
	var res []NowPlayingInfo
	playMap.Range(func(playerId, value interface{}) bool {
		info := value.(NowPlayingInfo)
		if time.Since(info.Start) < nowPlayingExpire {
			res = append(res, info)
		}
		return true
	})
	sort.Slice(res, func(i, j int) bool {
		return res[i].Start.After(res[j].Start)
	})
	return res, nil
}

func (s *scrobbler) Submit(ctx context.Context, playerId int, trackId string, playTime time.Time) error {
	panic("implement me")
}

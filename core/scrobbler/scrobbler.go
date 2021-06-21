package scrobbler

import (
	"context"
	"sort"
	"time"

	"github.com/ReneKroon/ttlcache/v2"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/singleton"
)

const nowPlayingExpire = 60 * time.Minute

type NowPlayingInfo struct {
	TrackID    string
	Start      time.Time
	Username   string
	PlayerId   string
	PlayerName string
}

type Scrobbler interface {
	NowPlaying(ctx context.Context, playerId string, playerName string, trackId string) error
	GetNowPlaying(ctx context.Context) ([]NowPlayingInfo, error)
	Submit(ctx context.Context, playerId int, trackId string, playTime time.Time) error
}

type scrobbler struct {
	ds      model.DataStore
	playMap *ttlcache.Cache
}

func GetInstance(ds model.DataStore) Scrobbler {
	instance := singleton.Get(scrobbler{}, func() interface{} {
		m := ttlcache.NewCache()
		m.SkipTTLExtensionOnHit(true)
		_ = m.SetTTL(nowPlayingExpire)
		return &scrobbler{ds: ds, playMap: m}
	})
	return instance.(*scrobbler)
}

func (s *scrobbler) NowPlaying(ctx context.Context, playerId string, playerName string, trackId string) error {
	username, _ := request.UsernameFrom(ctx)
	info := NowPlayingInfo{
		TrackID:    trackId,
		Start:      time.Now(),
		Username:   username,
		PlayerId:   playerId,
		PlayerName: playerName,
	}
	_ = s.playMap.Set(playerId, info)
	return nil
}

func (s *scrobbler) GetNowPlaying(ctx context.Context) ([]NowPlayingInfo, error) {
	var res []NowPlayingInfo
	for _, playerId := range s.playMap.GetKeys() {
		value, err := s.playMap.Get(playerId)
		if err != nil {
			continue
		}
		info := value.(NowPlayingInfo)
		res = append(res, info)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Start.After(res[j].Start)
	})
	return res, nil
}

func (s *scrobbler) Submit(ctx context.Context, playerId int, trackId string, playTime time.Time) error {
	panic("implement me")
}

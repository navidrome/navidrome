package persistence

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/deluan/gosonic/engine"
)

var (
	nowPlayingKeyName = []byte("nowplaying")
)

type nowPlayingRepository struct {
	ledisRepository
}

func NewNowPlayingRepository() engine.NowPlayingRepository {
	r := &nowPlayingRepository{}
	r.init("nowplaying", &engine.NowPlayingInfo{})
	return r
}

func (r *nowPlayingRepository) Add(id string) error {
	if id == "" {
		return errors.New("Id is required")
	}
	m := &engine.NowPlayingInfo{TrackId: id, Start: time.Now()}

	h, err := json.Marshal(m)
	if err != nil {
		return err
	}
	err = Db().Set(nowPlayingKeyName, []byte(h))
	if err != nil {
		return err
	}
	_, err = Db().Expire(nowPlayingKeyName, int64(engine.NowPlayingExpire.Seconds()))
	return err
}

var _ engine.NowPlayingRepository = (*nowPlayingRepository)(nil)

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

func (r *nowPlayingRepository) Set(id string) error {
	if id == "" {
		return errors.New("Id is required")
	}
	m := &engine.NowPlayingInfo{TrackId: id, Start: time.Now()}

	h, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return Db().SetEX(nowPlayingKeyName, int64(engine.NowPlayingExpire.Seconds()), []byte(h))
}

var _ engine.NowPlayingRepository = (*nowPlayingRepository)(nil)

package persistence

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/deluan/gosonic/engine"
)

var (
	nowPlayingKeyPrefix = []byte("nowplaying")
)

type nowPlayingRepository struct {
	ledisRepository
}

func NewNowPlayingRepository() engine.NowPlayingRepository {
	r := &nowPlayingRepository{}
	r.init("nowplaying", &engine.NowPlayingInfo{})
	return r
}

func nowPlayingKeyName(playerId int) string {
	return fmt.Sprintf("%s:%d", nowPlayingKeyPrefix, playerId)
}

func (r *nowPlayingRepository) Enqueue(playerId int, playerName, id, username string) error {
	m := &engine.NowPlayingInfo{TrackId: id, Username: username, Start: time.Now(), PlayerId: playerId, PlayerName: playerName}

	h, err := json.Marshal(m)
	if err != nil {
		return err
	}

	keyName := []byte(nowPlayingKeyName(playerId))

	_, err = Db().LPush(keyName, []byte(h))
	Db().LExpire(keyName, int64(engine.NowPlayingExpire.Seconds()))
	return err
}

func (r *nowPlayingRepository) Head(playerId int) (*engine.NowPlayingInfo, error) {
	keyName := []byte(nowPlayingKeyName(playerId))

	val, err := Db().LIndex(keyName, 0)
	if err != nil {
		return nil, err
	}
	info := &engine.NowPlayingInfo{}
	err = json.Unmarshal(val, info)
	if err != nil {
		return nil, nil
	}
	return info, nil
}

// TODO Will not work for multiple players
func (r *nowPlayingRepository) GetAll() ([]*engine.NowPlayingInfo, error) {
	np, err := r.Head(1)
	return []*engine.NowPlayingInfo{np}, err
}

func (r *nowPlayingRepository) Dequeue(playerId int) (*engine.NowPlayingInfo, error) {
	keyName := []byte(nowPlayingKeyName(playerId))

	val, err := Db().RPop(keyName)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, nil
	}
	info := &engine.NowPlayingInfo{}
	err = json.Unmarshal(val, info)
	if err != nil {
		return nil, nil
	}
	return info, nil
}

var _ engine.NowPlayingRepository = (*nowPlayingRepository)(nil)

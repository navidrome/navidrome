package persistence

import (
	"encoding/json"
	"fmt"

	"github.com/cloudsonic/sonic-server/engine"
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

func (r *nowPlayingRepository) Enqueue(info *engine.NowPlayingInfo) error {
	h, err := json.Marshal(info)
	if err != nil {
		return err
	}

	keyName := []byte(nowPlayingKeyName(info.PlayerId))

	_, err = Db().LPush(keyName, []byte(h))
	Db().LExpire(keyName, int64(engine.NowPlayingExpire.Seconds()))
	return err
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
	return r.parseInfo(val)
}

func (r *nowPlayingRepository) Head(playerId int) (*engine.NowPlayingInfo, error) {
	keyName := []byte(nowPlayingKeyName(playerId))

	val, err := Db().LIndex(keyName, 0)
	if err != nil {
		return nil, err
	}
	return r.parseInfo(val)
}

func (r *nowPlayingRepository) Tail(playerId int) (*engine.NowPlayingInfo, error) {
	keyName := []byte(nowPlayingKeyName(playerId))

	val, err := Db().LIndex(keyName, -1)
	if err != nil {
		return nil, err
	}
	return r.parseInfo(val)
}

func (r *nowPlayingRepository) Count(playerId int) (int64, error) {
	keyName := []byte(nowPlayingKeyName(playerId))
	return Db().LLen(keyName)
}

// TODO Will not work for multiple players
func (r *nowPlayingRepository) GetAll() ([]*engine.NowPlayingInfo, error) {
	np, err := r.Head(1)
	if np == nil || err != nil {
		return nil, err
	}
	return []*engine.NowPlayingInfo{np}, err
}

func (r *nowPlayingRepository) parseInfo(val []byte) (*engine.NowPlayingInfo, error) {
	info := &engine.NowPlayingInfo{}
	err := json.Unmarshal(val, info)
	if err != nil {
		return nil, nil
	}
	return info, nil
}

var _ engine.NowPlayingRepository = (*nowPlayingRepository)(nil)

package persistence

import (
	"errors"
	"time"

	"github.com/deluan/gosonic/engine"
)

type nowPlayingRepository struct {
	ledisRepository
}

func NewNowPlayingRepository() engine.NowPlayingRepository {
	r := &nowPlayingRepository{}
	r.init("nnowplaying", &engine.NowPlayingInfo{})
	return r
}

func (r *nowPlayingRepository) Add(id string) error {
	if id == "" {
		return errors.New("Id is required")
	}
	m := &engine.NowPlayingInfo{TrackId: id, Start: time.Now()}
	return r.saveOrUpdate(m.TrackId, m)
}

var _ engine.NowPlayingRepository = (*nowPlayingRepository)(nil)

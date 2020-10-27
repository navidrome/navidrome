package core

import (
	"container/list"
	"sync"
	"time"
)

const NowPlayingExpire = 60 * time.Minute

type NowPlayingInfo struct {
	TrackID    string
	Start      time.Time
	Username   string
	PlayerId   int
	PlayerName string
}

// This repo must have the semantics of a FIFO queue, for each playerId
type NowPlaying interface {
	// Insert at the head of the queue
	Enqueue(*NowPlayingInfo) error

	// Returns all heads from all playerIds
	GetAll() ([]*NowPlayingInfo, error)
}

var playerMap = sync.Map{}

type nowPlayingRepository struct{}

func NewNowPlayingRepository() NowPlaying {
	r := &nowPlayingRepository{}
	return r
}

func (r *nowPlayingRepository) Enqueue(info *NowPlayingInfo) error {
	l := r.getList(info.PlayerId)
	l.PushFront(info)
	return nil
}

func (r *nowPlayingRepository) GetAll() ([]*NowPlayingInfo, error) {
	var all []*NowPlayingInfo
	playerMap.Range(func(playerId, l interface{}) bool {
		ll := l.(*list.List)
		e := checkExpired(ll, ll.Front)
		if e != nil {
			all = append(all, e.Value.(*NowPlayingInfo))
		}
		return true
	})
	return all, nil
}

func (r *nowPlayingRepository) getList(id int) *list.List {
	l, _ := playerMap.LoadOrStore(id, list.New())
	return l.(*list.List)
}

func (r *nowPlayingRepository) dequeue(playerId int) (*NowPlayingInfo, error) {
	l := r.getList(playerId)
	e := checkExpired(l, l.Back)
	if e == nil {
		return nil, nil
	}
	l.Remove(e)
	return e.Value.(*NowPlayingInfo), nil
}

func (r *nowPlayingRepository) head(playerId int) (*NowPlayingInfo, error) {
	l := r.getList(playerId)
	e := checkExpired(l, l.Front)
	if e == nil {
		return nil, nil
	}
	return e.Value.(*NowPlayingInfo), nil
}

func (r *nowPlayingRepository) tail(playerId int) (*NowPlayingInfo, error) {
	l := r.getList(playerId)
	e := checkExpired(l, l.Back)
	if e == nil {
		return nil, nil
	}
	return e.Value.(*NowPlayingInfo), nil
}

func (r *nowPlayingRepository) count(playerId int) (int64, error) {
	l := r.getList(playerId)
	return int64(l.Len()), nil
}

func checkExpired(l *list.List, f func() *list.Element) *list.Element {
	for {
		e := f()
		if e == nil {
			return nil
		}
		start := e.Value.(*NowPlayingInfo).Start
		if time.Since(start) < NowPlayingExpire {
			return e
		}
		l.Remove(e)
	}
}

package engine

import "time"

const NowPlayingExpire = time.Duration(30) * time.Minute

type NowPlayingInfo struct {
	TrackId string
	Start   time.Time
}

type NowPlayingRepository interface {
	Set(trackId string) error
}

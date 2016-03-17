package engine

import "time"

const NowPlayingExpire = time.Duration(60) * time.Minute

type NowPlayingInfo struct {
	TrackId    string
	Start      time.Time
	Username   string
	PlayerId   int
	PlayerName string
}

type NowPlayingRepository interface {
	Set(trackId, username string, playerId int, playerName string) error
	Clear(playerId int) (*NowPlayingInfo, error)
	GetAll() (*[]NowPlayingInfo, error)
}

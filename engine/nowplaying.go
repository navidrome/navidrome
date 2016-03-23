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

// This repo has the semantics of a FIFO queue, for each playerId
type NowPlayingRepository interface {
	// Insert at the head of the queue
	Enqueue(playerId int, playerName string, trackId, username string) error

	// Returns the element at the head of the queue (last inserted one)
	Head(playerId int) (*NowPlayingInfo, error)

	// Removes and returns the element at the end of the queue
	Dequeue(playerId int) (*NowPlayingInfo, error)

	// Returns all heads from all playerIds
	GetAll() ([]*NowPlayingInfo, error)
}

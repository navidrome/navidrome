package domain

import "time"

const NowPlayingExpire = 60 * time.Minute

type NowPlayingInfo struct {
	TrackID    string
	Start      time.Time
	Username   string
	PlayerId   int
	PlayerName string
}

// This repo must have the semantics of a FIFO queue, for each playerId
type NowPlayingRepository interface {
	// Insert at the head of the queue
	Enqueue(*NowPlayingInfo) error

	// Removes and returns the element at the end of the queue
	Dequeue(playerId int) (*NowPlayingInfo, error)

	// Returns the element at the head of the queue (last inserted one)
	Head(playerId int) (*NowPlayingInfo, error)

	// Returns the element at the end of the queue (first inserted one)
	Tail(playerId int) (*NowPlayingInfo, error)

	// Size of the queue for the playerId
	Count(playerId int) (int64, error)

	// Returns all heads from all playerIds
	GetAll() ([]*NowPlayingInfo, error)
}

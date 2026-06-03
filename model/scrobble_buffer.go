package model

import "time"

type ScrobbleEntry struct {
	ID          string
	Service     string
	UserID      string
	PlayTime    time.Time
	EnqueueTime time.Time
	MediaFileID string
	MediaFile
	Client       string
	Source       string
	Origin       string
	PlaybackMode string
}

type ScrobbleEntries []ScrobbleEntry

type ScrobbleBufferRepository interface {
	UserIDs(service string) ([]string, error)
	Enqueue(service, userId, mediaFileId string, playTime time.Time, client, source, origin, playbackMode string) error
	Next(service string, userId string) (*ScrobbleEntry, error)
	Dequeue(entry *ScrobbleEntry) error
	Length() (int64, error)
}

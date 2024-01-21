package model

import "time"

type ScrobbleEntry struct {
	MediaFile
	Service     string
	UserID      string `structs:"user_id"`
	PlayTime    time.Time
	EnqueueTime time.Time
}

type ScrobbleEntries []ScrobbleEntry

type ScrobbleBufferRepository interface {
	UserIDs(service string) ([]string, error)
	Enqueue(service, userId, mediaFileId string, playTime time.Time) error
	Next(service string, userId string) (*ScrobbleEntry, error)
	Dequeue(entry *ScrobbleEntry) error
	Length() (int64, error)
}

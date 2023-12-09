package model

import "time"

type StarEntry struct {
	MediaFile
	Service     string
	UserID      string `structs:"user_id" orm:"column(user_id)"`
	IsStar      bool
	EnqueueTime time.Time
}

type StarEntries []StarEntry

type StarBufferRepository interface {
	UserIDs(service string) ([]string, error)
	TryUpdate(service, userId, mediaFileId string, isStar bool) (bool, error)
	Enqueue(service, userId, mediaFileId string, isStar bool) error
	Next(service string, userId string) (*StarEntry, error)
	Dequeue(entry *StarEntry) error
	Length() (int64, error)
}
